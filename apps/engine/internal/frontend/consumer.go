package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConsumer struct {
	client  *redis.Client
	service *Service
	logger  *slog.Logger
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
	return &RedisConsumer{
		client:  client,
		service: service,
		logger:  logger,
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
	err := c.client.XGroupCreateMkStream(ctx, streamKey, groupName, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		c.logger.Error("failed to create consumer group", slog.String("error", err.Error()))
		return
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
				if err != redis.Nil {
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
		WorkflowID:   fmt.Sprintf("wf-%d", job.WorkflowID),
		WorkflowType: "linkflow-workflow",
		TaskQueue:    "default-queue",
		Input:        []byte(payloadStr), // Pass the whole payload as input
		RequestID:    job.JobID,
	}

	_, err := c.service.StartWorkflowExecution(ctx, req)
	if err != nil {
		c.logger.Error("failed to start workflow", slog.String("error", err.Error()))
		// TODO: Handle retry logic or DLQ
	} else {
		c.logger.Info("started workflow execution", slog.String("job_id", job.JobID))
	}

	c.ack(ctx, stream, group, msg.ID)
}

func (c *RedisConsumer) ack(ctx context.Context, stream, group, id string) {
	c.client.XAck(ctx, stream, group, id)
}
