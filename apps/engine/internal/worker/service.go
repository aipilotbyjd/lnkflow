package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/linkflow/engine/internal/worker/adapter"
	"github.com/linkflow/engine/internal/worker/executor"
	"github.com/linkflow/engine/internal/worker/poller"
	"github.com/linkflow/engine/internal/worker/retry"

	commonv1 "github.com/linkflow/engine/api/gen/linkflow/common/v1"
	historyv1 "github.com/linkflow/engine/api/gen/linkflow/history/v1"
)

type Service struct {
	historyClient *adapter.HistoryClient
	matchingConn  *grpc.ClientConn
	executors     map[string]executor.Executor
	taskPollers   []*poller.Poller
	retryPolicy   *retry.Policy
	callbackHTTP  *http.Client
	callbackKey   string
	logger        *slog.Logger
	wg            sync.WaitGroup
	stopCh        chan struct{}

	mu      sync.RWMutex
	running bool
}

type Config struct {
	TaskQueues      []string
	NumPollers      int
	Identity        string
	MatchingAddr    string
	PollInterval    time.Duration
	RetryPolicy     *retry.Policy
	CallbackKey     string
	CallbackTimeout time.Duration
	Logger          *slog.Logger
	HistoryClient   *adapter.HistoryClient
}

// NewService creates a new worker service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.RetryPolicy == nil {
		cfg.RetryPolicy = retry.DefaultPolicy()
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.NumPollers <= 0 {
		cfg.NumPollers = 1
	}
	if cfg.CallbackTimeout <= 0 {
		cfg.CallbackTimeout = 10 * time.Second
	}
	if cfg.MatchingAddr == "" {
		return nil, fmt.Errorf("matching service address is required")
	}

	// Establish gRPC connection with proper options
	conn, err := grpc.NewClient(
		cfg.MatchingAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		cfg.Logger.Error("failed to connect to matching service", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to connect to matching service: %w", err)
	}

	client := adapter.NewMatchingClient(conn)

	var pollers []*poller.Poller
	for _, queue := range cfg.TaskQueues {
		for i := 0; i < cfg.NumPollers; i++ {
			identity := cfg.Identity
			if cfg.NumPollers > 1 {
				identity = fmt.Sprintf("%s-%d", cfg.Identity, i+1)
			}

			p := poller.New(poller.Config{
				Client:       client,
				TaskQueue:    queue,
				Identity:     identity,
				PollInterval: cfg.PollInterval,
				Logger:       cfg.Logger,
			})
			pollers = append(pollers, p)
		}
	}

	svc := &Service{
		historyClient: cfg.HistoryClient,
		matchingConn:  conn,
		executors:     make(map[string]executor.Executor),
		taskPollers:   pollers,
		retryPolicy:   cfg.RetryPolicy,
		callbackHTTP: &http.Client{
			Timeout: cfg.CallbackTimeout,
		},
		callbackKey: cfg.CallbackKey,
		logger:      cfg.Logger,
		stopCh:      make(chan struct{}),
	}

	for _, p := range pollers {
		p.SetHandler(svc.handleTask)
	}

	return svc, nil
}

func (s *Service) RegisterExecutor(exec executor.Executor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executors[exec.NodeType()] = exec
	s.logger.Info("registered executor", slog.String("node_type", exec.NodeType()))
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	for _, p := range s.taskPollers {
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("failed to start task poller: %w", err)
		}
	}

	s.logger.Info("worker service started")
	return nil
}

func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("service not running")
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	for _, p := range s.taskPollers {
		p.Stop()
	}
	s.wg.Wait()

	if s.matchingConn != nil {
		if err := s.matchingConn.Close(); err != nil {
			s.logger.Warn("failed to close matching connection", slog.String("error", err.Error()))
		}
	}

	s.logger.Info("worker service stopped")
	return nil
}

func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Service) handleTask(ctx context.Context, task *poller.Task) (*poller.TaskResult, error) {
	s.wg.Add(1)
	defer s.wg.Done()

	// Dispatch based on task type (Workflow vs Activity)
	// Currently the poller returns a generic task. We should infer type from task.NodeType or similar.
	// The poller.Task struct has NodeType.
	if task.NodeType == "workflow" {
		return s.processWorkflowTask(ctx, task)
	}
	return s.processActivityTask(ctx, task)
}

func (s *Service) processWorkflowTask(ctx context.Context, task *poller.Task) (*poller.TaskResult, error) {
	s.logger.Info("processing workflow task", slog.String("workflow_id", task.WorkflowID))
	startedAt := time.Now()
	jobPayload, payloadErr := s.loadJobPayload(ctx, task)
	if payloadErr != nil {
		s.logger.Warn("failed to load callback payload",
			slog.String("workflow_id", task.WorkflowID),
			slog.String("run_id", task.RunID),
			slog.String("error", payloadErr.Error()),
		)
	}

	// Get Workflow Executor
	exec, ok := s.executors["workflow"]
	if !ok {
		return nil, fmt.Errorf("workflow executor not found")
	}

	req := &executor.ExecuteRequest{
		NodeType:   "workflow",
		WorkflowID: task.WorkflowID,
		RunID:      task.RunID,
		Namespace:  task.Namespace,
		Input:      task.Input,
		Attempt:    task.Attempt,
		Timeout:    30 * time.Second,
	}

	resp, err := exec.Execute(ctx, req)
	if err != nil {
		s.logger.Error("workflow execution failed", slog.String("error", err.Error()))
		// Respond failed
		s.historyClient.RespondWorkflowTaskFailed(ctx, &historyv1.RespondWorkflowTaskFailedRequest{
			Namespace: task.Namespace,
			WorkflowExecution: &commonv1.WorkflowExecution{
				WorkflowId: task.WorkflowID,
				RunId:      task.RunID,
			},
			TaskToken: task.ScheduledEventID,
			Failure: &commonv1.Failure{
				Message: err.Error(),
			},
		})
		s.sendLegacyCallback(jobPayload, "failed", time.Since(startedAt), map[string]interface{}{
			"message": err.Error(),
		})
		return nil, err
	}

	// ExecuteResponse.Output now contains the Commands (marshaled)
	var commands []*historyv1.Command
	if err := json.Unmarshal(resp.Output, &commands); err != nil {
		s.logger.Error("failed to unmarshal workflow commands", slog.String("error", err.Error()))
		return nil, err
	}

	_, err = s.historyClient.RespondWorkflowTaskCompleted(ctx, &historyv1.RespondWorkflowTaskCompletedRequest{
		Namespace: task.Namespace,
		WorkflowExecution: &commonv1.WorkflowExecution{
			WorkflowId: task.WorkflowID,
			RunId:      task.RunID,
		},
		TaskToken: task.ScheduledEventID,
		Commands:  commands,
	})
	if err != nil {
		s.logger.Error("failed to respond workflow task completed", slog.String("error", err.Error()))
		s.sendLegacyCallback(jobPayload, "failed", time.Since(startedAt), map[string]interface{}{
			"message": err.Error(),
		})
		return nil, err
	}

	status, callbackErr := callbackStatusFromCommands(commands)
	if status != "" {
		s.sendLegacyCallback(jobPayload, status, time.Since(startedAt), callbackErr)
	}

	return &poller.TaskResult{TaskID: task.TaskID}, nil
}

func (s *Service) processActivityTask(ctx context.Context, task *poller.Task) (*poller.TaskResult, error) {
	s.logger.Info("processing activity task", slog.String("node_type", task.NodeType), slog.String("node_id", task.NodeID))

	if task.NodeType == "" || task.NodeID == "" || len(task.Input) == 0 {
		if err := s.hydrateActivityTaskFromHistory(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to hydrate activity task: %w", err)
		}
	}

	s.mu.RLock()
	exec, ok := s.executors[task.NodeType]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("executor not found for type: %s", task.NodeType)
	}

	req := &executor.ExecuteRequest{
		NodeType:   task.NodeType,
		NodeID:     task.NodeID,
		WorkflowID: task.WorkflowID,
		RunID:      task.RunID,
		Namespace:  task.Namespace,
		Config:     task.Config,
		Input:      task.Input,
		Attempt:    task.Attempt,
		Timeout:    time.Duration(task.TimeoutSec) * time.Second,
	}

	resp, err := exec.Execute(ctx, req)

	// Handle execution result
	if err != nil {
		// System error (crash, timeout)
		s.historyClient.RespondActivityTaskFailed(ctx, &historyv1.RespondActivityTaskFailedRequest{
			Namespace: task.Namespace,
			WorkflowExecution: &commonv1.WorkflowExecution{
				WorkflowId: task.WorkflowID,
				RunId:      task.RunID,
			},
			ScheduledEventId: task.ScheduledEventID,
			Failure: &commonv1.Failure{
				Message:     err.Error(),
				FailureType: commonv1.FailureType_FAILURE_TYPE_ACTIVITY,
			},
		})
		return &poller.TaskResult{Error: err.Error()}, err
	}

	if resp.Error != nil {
		// Logical error (API failure, etc.)
		s.historyClient.RespondActivityTaskFailed(ctx, &historyv1.RespondActivityTaskFailedRequest{
			Namespace: task.Namespace,
			WorkflowExecution: &commonv1.WorkflowExecution{
				WorkflowId: task.WorkflowID,
				RunId:      task.RunID,
			},
			ScheduledEventId: task.ScheduledEventID,
			Failure: &commonv1.Failure{
				Message:     resp.Error.Message,
				FailureType: commonv1.FailureType_FAILURE_TYPE_APPLICATION,
			},
		})
		return &poller.TaskResult{Error: resp.Error.Message}, nil
	}

	// Success
	_, err = s.historyClient.RespondActivityTaskCompleted(ctx, &historyv1.RespondActivityTaskCompletedRequest{
		Namespace: task.Namespace,
		WorkflowExecution: &commonv1.WorkflowExecution{
			WorkflowId: task.WorkflowID,
			RunId:      task.RunID,
		},
		ScheduledEventId: task.ScheduledEventID,
		Result: &commonv1.Payloads{
			Payloads: []*commonv1.Payload{{Data: resp.Output}},
		},
	})

	return &poller.TaskResult{Output: resp.Output}, err
}

func (s *Service) hydrateActivityTaskFromHistory(ctx context.Context, task *poller.Task) error {
	historyResp, err := s.historyClient.GetHistory(ctx, task.Namespace, task.WorkflowID, task.RunID)
	if err != nil {
		return err
	}

	events := historyResp.GetHistory().GetEvents()
	for _, event := range events {
		if event.GetEventId() != task.ScheduledEventID {
			continue
		}
		if event.GetEventType() != commonv1.EventType_EVENT_TYPE_NODE_SCHEDULED {
			continue
		}

		attr := event.GetNodeScheduledAttributes()
		if attr == nil {
			continue
		}

		task.NodeID = attr.GetNodeId()
		task.NodeType = attr.GetNodeType()

		if input := attr.GetInput(); input != nil && len(input.GetPayloads()) > 0 {
			raw := input.GetPayloads()[0].GetData()
			task.Input = raw

			var envelope struct {
				Input  json.RawMessage `json:"input"`
				Config json.RawMessage `json:"config"`
				NodeID string          `json:"node_id"`
				Type   string          `json:"node_type"`
			}

			if err := json.Unmarshal(raw, &envelope); err == nil && (len(envelope.Input) > 0 || len(envelope.Config) > 0) {
				if len(envelope.Input) > 0 {
					task.Input = envelope.Input
				}
				if len(envelope.Config) > 0 {
					task.Config = envelope.Config
				}
				if envelope.NodeID != "" {
					task.NodeID = envelope.NodeID
				}
				if envelope.Type != "" {
					task.NodeType = envelope.Type
				}
			}
		}

		if task.NodeType == "" {
			return fmt.Errorf("missing node_type for scheduled_event_id=%d", task.ScheduledEventID)
		}

		return nil
	}

	return fmt.Errorf("scheduled event %d not found", task.ScheduledEventID)
}

func (s *Service) loadJobPayload(ctx context.Context, task *poller.Task) (*executor.JobPayload, error) {
	historyResp, err := s.historyClient.GetHistory(ctx, task.Namespace, task.WorkflowID, task.RunID)
	if err != nil {
		return nil, err
	}

	events := historyResp.GetHistory().GetEvents()
	for _, event := range events {
		if event.GetEventType() != commonv1.EventType_EVENT_TYPE_EXECUTION_STARTED {
			continue
		}

		attr := event.GetExecutionStartedAttributes()
		if attr == nil || attr.GetInput() == nil || len(attr.GetInput().GetPayloads()) == 0 {
			continue
		}

		payloadBytes := attr.GetInput().GetPayloads()[0].GetData()

		var payload executor.JobPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, err
		}

		return &payload, nil
	}

	return nil, fmt.Errorf("execution started payload not found")
}

func callbackStatusFromCommands(commands []*historyv1.Command) (string, map[string]interface{}) {
	status := ""
	var callbackErr map[string]interface{}

	for _, cmd := range commands {
		switch cmd.GetCommandType() {
		case historyv1.CommandType_COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION:
			if status == "" {
				status = "completed"
			}
		case historyv1.CommandType_COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION:
			status = "failed"
			callbackErr = map[string]interface{}{
				"message": "workflow execution failed",
			}
			if attr := cmd.GetFailWorkflowExecutionAttributes(); attr != nil && attr.GetFailure() != nil && attr.GetFailure().GetMessage() != "" {
				callbackErr["message"] = attr.GetFailure().GetMessage()
			}
		}
	}

	return status, callbackErr
}

func (s *Service) sendLegacyCallback(payload *executor.JobPayload, status string, duration time.Duration, callbackErr map[string]interface{}) {
	if payload == nil || payload.CallbackURL == "" || payload.JobID == "" || payload.CallbackToken == "" || payload.ExecutionID == 0 {
		return
	}
	if status != "completed" && status != "failed" {
		return
	}

	body := map[string]interface{}{
		"job_id":         payload.JobID,
		"callback_token": payload.CallbackToken,
		"execution_id":   payload.ExecutionID,
		"status":         status,
		"duration_ms":    duration.Milliseconds(),
	}
	if callbackErr != nil {
		body["error"] = callbackErr
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		s.logger.Error("failed to marshal callback payload", slog.String("error", err.Error()))
		return
	}

	for attempt := 1; attempt <= 3; attempt++ {
		reqCtx, cancel := context.WithTimeout(context.Background(), s.callbackHTTP.Timeout)
		err = s.postLegacyCallback(reqCtx, payload.CallbackURL, bodyBytes)
		cancel()

		if err == nil {
			return
		}

		s.logger.Warn("failed to send workflow callback",
			slog.String("job_id", payload.JobID),
			slog.String("status", status),
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()),
		)

		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
}

func (s *Service) postLegacyCallback(ctx context.Context, callbackURL string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LinkFlow-Timestamp", time.Now().UTC().Format(time.RFC3339))
	if s.callbackKey != "" {
		req.Header.Set("X-LinkFlow-Signature", signPayload(body, s.callbackKey))
	}

	resp, err := s.callbackHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("callback returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func signPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
