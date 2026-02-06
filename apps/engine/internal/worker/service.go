package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	logger        *slog.Logger
	wg            sync.WaitGroup
	stopCh        chan struct{}

	mu      sync.RWMutex
	running bool
}

type Config struct {
	TaskQueues    []string
	Identity      string
	MatchingAddr  string
	PollInterval  time.Duration
	RetryPolicy   *retry.Policy
	Logger        *slog.Logger
	HistoryClient *adapter.HistoryClient
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
		p := poller.New(poller.Config{
			Client:       client,
			TaskQueue:    queue,
			Identity:     cfg.Identity,
			PollInterval: cfg.PollInterval,
			Logger:       cfg.Logger,
		})
		pollers = append(pollers, p)
	}

	svc := &Service{
		historyClient: cfg.HistoryClient,
		matchingConn:  conn,
		executors:     make(map[string]executor.Executor),
		taskPollers:   pollers,
		retryPolicy:   cfg.RetryPolicy,
		logger:        cfg.Logger,
		stopCh:        make(chan struct{}),
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
		return nil, err
	}

	return &poller.TaskResult{TaskID: task.TaskID}, nil
}

func (s *Service) processActivityTask(ctx context.Context, task *poller.Task) (*poller.TaskResult, error) {
	s.logger.Info("processing activity task", slog.String("node_type", task.NodeType), slog.String("node_id", task.NodeID))

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
