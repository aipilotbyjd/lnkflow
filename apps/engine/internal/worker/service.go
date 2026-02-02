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
	shutdownOnce  sync.Once

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

// NewService creates a new worker service. Returns an error if configuration is invalid
// or if the connection to the matching service fails.
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
		return ErrServiceAlreadyStart
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
	var stopErr error
	s.shutdownOnce.Do(func() {
		s.mu.Lock()
		if !s.running {
			s.mu.Unlock()
			stopErr = ErrServiceNotRunning
			return
		}
		s.running = false
		close(s.stopCh)
		s.mu.Unlock()

		// Stop all pollers
		for _, p := range s.taskPollers {
			p.Stop()
		}

		// Wait for in-flight tasks to complete
		s.wg.Wait()

		// Close the gRPC connection to matching service
		if s.matchingConn != nil {
			if err := s.matchingConn.Close(); err != nil {
				s.logger.Warn("failed to close matching connection", slog.String("error", err.Error()))
			}
		}

		s.logger.Info("worker service stopped")
	})
	return stopErr
}

func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Service) handleTask(ctx context.Context, task *poller.Task) (*poller.TaskResult, error) {
	s.wg.Add(1)
	defer s.wg.Done()

	workerTask := &Task{
		TaskID:           task.TaskID,
		WorkflowID:       task.WorkflowID,
		RunID:            task.RunID,
		Namespace:        task.Namespace,
		NodeType:         task.NodeType,
		NodeID:           task.NodeID,
		Config:           task.Config,
		Input:            task.Input,
		Attempt:          task.Attempt,
		TimeoutSec:       task.TimeoutSec,
		ScheduledEventID: task.ScheduledEventID,
	}

	result, err := s.ProcessTask(ctx, workerTask)
	if err != nil {
		return nil, err
	}

	return &poller.TaskResult{
		TaskID:    result.TaskID,
		Output:    result.Output,
		Error:     result.Error,
		ErrorType: result.ErrorType,
		Logs:      result.Logs,
	}, nil
}

func (s *Service) ProcessTask(ctx context.Context, task *Task) (*TaskResult, error) {
	s.logger.Info("processing task",
		slog.String("task_id", task.TaskID),
		slog.String("node_type", task.NodeType),
		slog.String("node_id", task.NodeID),
		slog.Int("attempt", int(task.Attempt)),
	)

	s.mu.RLock()
	exec, ok := s.executors[task.NodeType]
	s.mu.RUnlock()

	if !ok {
		return &TaskResult{
			TaskID:    task.TaskID,
			Error:     fmt.Sprintf("no executor registered for node type: %s", task.NodeType),
			ErrorType: ErrorTypeNonRetryable,
		}, ErrExecutorNotFound
	}

	timeout := time.Duration(task.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req := &executor.ExecuteRequest{
		NodeType:   task.NodeType,
		NodeID:     task.NodeID,
		WorkflowID: task.WorkflowID,
		RunID:      task.RunID,
		Namespace:  task.Namespace,
		Config:     task.Config,
		Input:      task.Input,
		Attempt:    task.Attempt,
		Timeout:    timeout,
	}

	resp, err := exec.Execute(execCtx, req)
	if err != nil {
		s.logger.Error("executor error",
			slog.String("task_id", task.TaskID),
			slog.String("error", err.Error()),
		)

		// Record Failure in History (if not workflow task)
		if task.NodeType != "workflow" && s.historyClient != nil {
			// We need next eventID? History service calculates it.
			// But wait, RecordEvent API takes an event with EventID?
			// history/grpc_server says RecordEvent returns the new EventID.
			// The passed event must have *some* ID? Or can be 0?
			// history/engine.go IncrementNextEventID uses state.
			// history/grpc_server.go protoEventToInternal maps it.
			// Let's check if we can pass 0.

			event := &historyv1.HistoryEvent{
				EventType: commonv1.EventType_EVENT_TYPE_NODE_FAILED,
				Attributes: &historyv1.HistoryEvent_NodeFailedAttributes{
					NodeFailedAttributes: &historyv1.NodeFailedEventAttributes{
						ScheduledEventId: task.ScheduledEventID,
						// StartedEventId: ... we don't track started event ID yet?
						// For now assume ScheduledEventId + 1? No.
						Failure: &commonv1.Failure{
							Message: err.Error(),
						},
					},
				},
			}
			// Determine namespace from task?
			namespace := task.Namespace
			if namespace == "" {
				namespace = "default"
			}

			_ = s.historyClient.RecordEvent(context.Background(), namespace, task.WorkflowID, task.RunID, event)
		}

		return &TaskResult{
			TaskID:    task.TaskID,
			Error:     err.Error(),
			ErrorType: ErrorTypeRetryable,
		}, err
	}

	result := &TaskResult{
		TaskID: task.TaskID,
		Output: resp.Output,
	}

	if resp.Error != nil {
		result.Error = resp.Error.Message
		result.ErrorType = resp.Error.Type

		if s.retryPolicy.ShouldRetry(task.Attempt, resp.Error.Type, resp.Error.Message) {
			delay := s.retryPolicy.NextRetryDelay(task.Attempt)
			s.logger.Info("task will be retried",
				slog.String("task_id", task.TaskID),
				slog.Int("attempt", int(task.Attempt)),
				slog.Duration("retry_delay", delay),
			)
		} else {
			// Record Logic Failure (Non-retryable or exhausted)
			if task.NodeType != "workflow" && s.historyClient != nil {
				event := &historyv1.HistoryEvent{
					EventType: commonv1.EventType_EVENT_TYPE_NODE_FAILED,
					Attributes: &historyv1.HistoryEvent_NodeFailedAttributes{
						NodeFailedAttributes: &historyv1.NodeFailedEventAttributes{
							ScheduledEventId: task.ScheduledEventID,
							Failure: &commonv1.Failure{
								Message: result.Error,
							},
						},
					},
				}
				namespace := task.Namespace
				if namespace == "" {
					namespace = "default"
				}
				_ = s.historyClient.RecordEvent(context.Background(), namespace, task.WorkflowID, task.RunID, event)
			}
		}
	} else {
		// Record Success
		if task.NodeType != "workflow" && s.historyClient != nil {
			event := &historyv1.HistoryEvent{
				EventType: commonv1.EventType_EVENT_TYPE_NODE_COMPLETED,
				Attributes: &historyv1.HistoryEvent_NodeCompletedAttributes{
					NodeCompletedAttributes: &historyv1.NodeCompletedEventAttributes{
						ScheduledEventId: task.ScheduledEventID,
						Result: &commonv1.Payloads{
							Payloads: []*commonv1.Payload{{Data: resp.Output}},
						},
					},
				},
			}
			namespace := task.Namespace
			if namespace == "" {
				namespace = "default"
			}

			err := s.historyClient.RecordEvent(context.Background(), namespace, task.WorkflowID, task.RunID, event)
			if err != nil {
				s.logger.Error("failed to record node completion", slog.String("error", err.Error()))
			}
		}
	}

	if len(resp.Logs) > 0 {
		logsJSON, _ := json.Marshal(resp.Logs)
		result.Logs = logsJSON
	}

	s.logger.Info("task processed",
		slog.String("task_id", task.TaskID),
		slog.Duration("duration", resp.Duration),
		slog.Bool("has_error", resp.Error != nil),
	)

	return result, nil
}

func (s *Service) GetExecutor(nodeType string) (executor.Executor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exec, ok := s.executors[nodeType]
	return exec, ok
}
