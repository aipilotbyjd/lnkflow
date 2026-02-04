package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultMaxRetries   = 3
	DefaultBaseDelay    = time.Second
	DefaultMaxDelay     = 30 * time.Second
	DefaultDLQStreamKey = "linkflow:jobs:dlq"
)

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

type ConsumerConfig struct {
	Retry        RetryConfig
	DLQStreamKey string
}

func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		Retry: RetryConfig{
			MaxRetries: DefaultMaxRetries,
			BaseDelay:  DefaultBaseDelay,
			MaxDelay:   DefaultMaxDelay,
		},
		DLQStreamKey: DefaultDLQStreamKey,
	}
}

type RedisConsumer struct {
	client  *redis.Client
	service *Service
	logger  *slog.Logger
	config  ConsumerConfig
}

type JobPayload struct {
	JobID         string                 `json:"job_id"`
	CallbackToken string                 `json:"callback_token"`
	ExecutionID   int                    `json:"execution_id"`
	WorkflowID    int                    `json:"workflow_id"`
	WorkspaceID   int                    `json:"workspace_id"`
	Partition     int                    `json:"partition"`
	Priority      string                 `json:"priority"`
	Workflow      map[string]interface{} `json:"workflow"`
	TriggerData   map[string]interface{} `json:"trigger_data"`
	Credentials   map[string]interface{} `json:"credentials"`
	Variables     map[string]interface{} `json:"variables"`
	CallbackURL   string                 `json:"callback_url"`
	ProgressURL   string                 `json:"progress_url"`
}

func NewRedisConsumer(client *redis.Client, service *Service, logger *slog.Logger) *RedisConsumer {
	return NewRedisConsumerWithConfig(client, service, logger, DefaultConsumerConfig())
}

func NewRedisConsumerWithConfig(client *redis.Client, service *Service, logger *slog.Logger, config ConsumerConfig) *RedisConsumer {
	return &RedisConsumer{
		client:  client,
		service: service,
		logger:  logger,
		config:  config,
	}
}

func (c *RedisConsumer) Start(ctx context.Context) {
	// Listen to all partitions (0-15)
	for i := 0; i < 16; i++ {
		go c.consumePartition(ctx, i)
	}
}

func (c *RedisConsumer) consumePartition(ctx context.Context, partition int) {
	streamKey := fmt.Sprintf("linkflow:jobs:partition:%d", partition)
	groupName := "engine-group"
	consumerName := fmt.Sprintf("engine-consumer-%d", partition)

	// Create consumer group
	for {
		err := c.client.XGroupCreateMkStream(ctx, streamKey, groupName, "$").Err()
		if err == nil {
			break
		}
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			break
		}

		c.logger.Error("failed to create consumer group, retrying...", slog.String("error", err.Error()))
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			continue
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    groupName,
				Consumer: consumerName,
				Streams:  []string{streamKey, ">"},
				Count:    1,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if !errors.Is(err, redis.Nil) {
					c.logger.Error("failed to read stream", slog.String("error", err.Error()))
					time.Sleep(time.Second)
				}
				continue
			}

			for _, stream := range streams {
				for _, msg := range stream.Messages {
					c.processMessage(ctx, msg, streamKey, groupName)
				}
			}
		}
	}
}

func (c *RedisConsumer) processMessage(ctx context.Context, msg redis.XMessage, stream, group string) {
	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		c.logger.Error("invalid payload format")
		c.ack(ctx, stream, group, msg.ID)
		return
	}

	var job JobPayload
	if err := json.Unmarshal([]byte(payloadStr), &job); err != nil {
		c.logger.Error("failed to unmarshal payload", slog.String("error", err.Error()))
		c.ack(ctx, stream, group, msg.ID)
		return
	}

	c.logger.Info("processing job", slog.String("job_id", job.JobID))

	// Map to StartWorkflowExecutionRequest
	req := &StartWorkflowExecutionRequest{
		Namespace:    fmt.Sprintf("workspace-%d", job.WorkspaceID),
		WorkflowID:   fmt.Sprintf("workflow-%d", job.WorkflowID),
		WorkflowType: "linkflow-workflow",
		TaskQueue:    fmt.Sprintf("workflows-%s", job.Priority),
		Input:        []byte(payloadStr), // Pass the whole payload as input
		RequestID:    job.JobID,
	}

	if err := c.executeWithRetry(ctx, req, &job, payloadStr, stream, group, msg.ID); err != nil {
		c.logger.Error("job failed after all retries, moved to DLQ",
			slog.String("job_id", job.JobID),
			slog.String("error", err.Error()),
		)
	}

	c.ack(ctx, stream, group, msg.ID)
}

func (c *RedisConsumer) executeWithRetry(ctx context.Context, req *StartWorkflowExecutionRequest, job *JobPayload, payloadStr, stream, _, msgID string) error {
	var lastErr error

	for attempt := 1; attempt <= c.config.Retry.MaxRetries; attempt++ {
		c.logger.Info("attempting to start workflow",
			slog.String("job_id", job.JobID),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", c.config.Retry.MaxRetries),
		)

		_, err := c.service.StartWorkflowExecution(ctx, req)
		if err == nil {
			c.logger.Info("started workflow execution",
				slog.String("job_id", job.JobID),
				slog.Int("attempts", attempt),
			)
			return nil
		}

		lastErr = err
		c.logger.Warn("workflow execution failed",
			slog.String("job_id", job.JobID),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", c.config.Retry.MaxRetries),
			slog.String("error", err.Error()),
		)

		if attempt < c.config.Retry.MaxRetries {
			delay := c.calculateBackoff(attempt)
			c.logger.Info("waiting before retry",
				slog.String("job_id", job.JobID),
				slog.Duration("delay", delay),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	if err := c.moveToDLQ(ctx, job, payloadStr, stream, msgID, lastErr); err != nil {
		c.logger.Error("failed to move message to DLQ",
			slog.String("job_id", job.JobID),
			slog.String("error", err.Error()),
		)
	}

	return lastErr
}

func (c *RedisConsumer) calculateBackoff(attempt int) time.Duration {
	delay := c.config.Retry.BaseDelay * time.Duration(1<<uint(attempt-1))
	if delay > c.config.Retry.MaxDelay {
		delay = c.config.Retry.MaxDelay
	}
	return delay
}

type DLQEntry struct {
	OriginalPayload string    `json:"original_payload"`
	OriginalStream  string    `json:"original_stream"`
	OriginalMsgID   string    `json:"original_msg_id"`
	JobID           string    `json:"job_id"`
	FailureReason   string    `json:"failure_reason"`
	AttemptCount    int       `json:"attempt_count"`
	FailedAt        time.Time `json:"failed_at"`
}

func (c *RedisConsumer) moveToDLQ(ctx context.Context, job *JobPayload, payloadStr, originalStream, originalMsgID string, lastErr error) error {
	entry := DLQEntry{
		OriginalPayload: payloadStr,
		OriginalStream:  originalStream,
		OriginalMsgID:   originalMsgID,
		JobID:           job.JobID,
		FailureReason:   lastErr.Error(),
		AttemptCount:    c.config.Retry.MaxRetries,
		FailedAt:        time.Now().UTC(),
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ entry: %w", err)
	}

	_, err = c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: c.config.DLQStreamKey,
		Values: map[string]interface{}{
			"payload": string(entryJSON),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add to DLQ stream: %w", err)
	}

	c.logger.Info("moved message to DLQ",
		slog.String("job_id", job.JobID),
		slog.String("dlq_stream", c.config.DLQStreamKey),
		slog.Int("attempt_count", c.config.Retry.MaxRetries),
		slog.String("failure_reason", lastErr.Error()),
	)

	return nil
}

func (c *RedisConsumer) ack(ctx context.Context, stream, group, id string) {
	c.client.XAck(ctx, stream, group, id)
}
