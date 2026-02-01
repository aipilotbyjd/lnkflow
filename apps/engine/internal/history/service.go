package history

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/linkflow/engine/internal/history/engine"
	"github.com/linkflow/engine/internal/history/shard"
	"github.com/linkflow/engine/internal/history/types"
)

var (
	ErrServiceNotRunning     = errors.New("history service is not running")
	ErrServiceAlreadyRunning = errors.New("history service is already running")
	ErrExecutionNotFound     = errors.New("execution not found")
	ErrEventNotFound         = errors.New("event not found")
	ErrOptimisticLock        = errors.New("optimistic lock failure")
)

// EventStore defines the interface for storing and retrieving history events.
type EventStore interface {
	AppendEvents(ctx context.Context, key types.ExecutionKey, events []*types.HistoryEvent, expectedVersion int64) error
	GetEvents(ctx context.Context, key types.ExecutionKey, firstEventID, lastEventID int64) ([]*types.HistoryEvent, error)
}

// MutableStateStore defines the interface for storing workflow mutable state.
type MutableStateStore interface {
	GetMutableState(ctx context.Context, key types.ExecutionKey) (*engine.MutableState, error)
	UpdateMutableState(ctx context.Context, key types.ExecutionKey, state *engine.MutableState, expectedVersion int64) error
}

// ShardController manages shard ownership and distribution.
type ShardController interface {
	GetShardForExecution(key types.ExecutionKey) (shard.Shard, error)
	GetShardIDForExecution(key types.ExecutionKey) int32
	Stop()
}

// Metrics provides hooks for observability.
type Metrics interface {
	RecordEventRecorded(eventType types.EventType)
	RecordEventRetrieved(count int)
	RecordServiceLatency(operation string, duration time.Duration)
}

// noopMetrics is a no-op implementation of Metrics.
type noopMetrics struct{}

func (noopMetrics) RecordEventRecorded(types.EventType)        {}
func (noopMetrics) RecordEventRetrieved(int)                   {}
func (noopMetrics) RecordServiceLatency(string, time.Duration) {}

// Service provides workflow history management capabilities.
type Service struct {
	shardController ShardController
	eventStore      EventStore
	stateStore      MutableStateStore
	historyEngine   *engine.Engine
	metrics         Metrics
	logger          *slog.Logger

	running bool
	mu      sync.RWMutex
}

// Config holds configuration for the history service.
type Config struct {
	ShardController ShardController
	EventStore      EventStore
	StateStore      MutableStateStore
	Logger          *slog.Logger
	Metrics         Metrics
}

// NewService creates a new history service with default config.
func NewService(
	shardController ShardController,
	eventStore EventStore,
	stateStore MutableStateStore,
	logger *slog.Logger,
) *Service {
	return NewServiceWithConfig(Config{
		ShardController: shardController,
		EventStore:      eventStore,
		StateStore:      stateStore,
		Logger:          logger,
	})
}

// NewServiceWithConfig creates a new history service with full configuration.
func NewServiceWithConfig(cfg Config) *Service {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	metrics := cfg.Metrics
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &Service{
		shardController: cfg.ShardController,
		eventStore:      cfg.EventStore,
		stateStore:      cfg.StateStore,
		historyEngine:   engine.NewEngine(cfg.Logger),
		metrics:         metrics,
		logger:          cfg.Logger,
		running:         false,
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrServiceAlreadyRunning
	}

	s.logger.Info("starting history service")
	s.running = true
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("stopping history service")

	if s.shardController != nil {
		s.shardController.Stop()
	}

	s.running = false
	return nil
}

func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Service) RecordEvent(ctx context.Context, key types.ExecutionKey, event *types.HistoryEvent) error {
	start := time.Now()
	defer func() {
		s.metrics.RecordServiceLatency("RecordEvent", time.Since(start))
	}()

	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return ErrServiceNotRunning
	}

	shard, err := s.shardController.GetShardForExecution(key)
	if err != nil {
		return err
	}

	s.logger.Debug("recording event",
		"shard_id", shard.GetID(),
		"namespace_id", key.NamespaceID,
		"workflow_id", key.WorkflowID,
		"run_id", key.RunID,
		"event_id", event.EventID,
		"event_type", event.EventType,
	)

	state, err := s.stateStore.GetMutableState(ctx, key)
	if err != nil {
		if errors.Is(err, ErrExecutionNotFound) {
			// Create new mutable state if it doesn't exist
			state = engine.NewMutableState(&types.ExecutionInfo{
				NamespaceID: key.NamespaceID,
				WorkflowID:  key.WorkflowID,
				RunID:       key.RunID,
			})
		} else {
			return err
		}
	}

	expectedVersion := state.DBVersion

	// Use the engine logic to validate and apply the event to the state
	if err := s.historyEngine.ProcessEvent(state, event); err != nil {
		return err
	}

	if err := s.eventStore.AppendEvents(ctx, key, []*types.HistoryEvent{event}, expectedVersion); err != nil {
		return err
	}

	state.DBVersion++

	if err := s.stateStore.UpdateMutableState(ctx, key, state, expectedVersion); err != nil {
		s.logger.Warn("failed to update mutable state after recording event",
			"error", err,
			"workflow_id", key.WorkflowID,
		)
		return err
	}

	s.metrics.RecordEventRecorded(event.EventType)
	return nil
}

func (s *Service) RecordEvents(ctx context.Context, key types.ExecutionKey, events []*types.HistoryEvent) error {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return ErrServiceNotRunning
	}

	if len(events) == 0 {
		return nil
	}

	_, err := s.shardController.GetShardForExecution(key)
	if err != nil {
		return err
	}

	state, err := s.stateStore.GetMutableState(ctx, key)
	if err != nil {
		if errors.Is(err, ErrExecutionNotFound) {
			state = engine.NewMutableState(&types.ExecutionInfo{
				NamespaceID: key.NamespaceID,
				WorkflowID:  key.WorkflowID,
				RunID:       key.RunID,
			})
		} else {
			return err
		}
	}

	expectedVersion := state.DBVersion

	// Apply all events
	for _, event := range events {
		if err := s.historyEngine.ProcessEvent(state, event); err != nil {
			return err
		}
	}

	if err := s.eventStore.AppendEvents(ctx, key, events, expectedVersion); err != nil {
		return err
	}

	state.DBVersion++

	if err := s.stateStore.UpdateMutableState(ctx, key, state, expectedVersion); err != nil {
		s.logger.Warn("failed to update mutable state", "error", err)
		return err
	}

	return nil
}

func (s *Service) GetHistory(ctx context.Context, key types.ExecutionKey, firstEventID, lastEventID int64) ([]*types.HistoryEvent, error) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return nil, ErrServiceNotRunning
	}

	if firstEventID <= 0 {
		firstEventID = 1
	}
	if lastEventID <= 0 {
		lastEventID = int64(^uint64(0) >> 1)
	}

	events, err := s.eventStore.GetEvents(ctx, key, firstEventID, lastEventID)
	if err != nil {
		return nil, err
	}
	s.metrics.RecordEventRetrieved(len(events))
	return events, nil
}

func (s *Service) GetMutableState(ctx context.Context, key types.ExecutionKey) (*engine.MutableState, error) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return nil, ErrServiceNotRunning
	}

	return s.stateStore.GetMutableState(ctx, key)
}

func (s *Service) GetShardForExecution(key types.ExecutionKey) (shard.Shard, error) {
	return s.shardController.GetShardForExecution(key)
}

func (s *Service) GetShardIDForExecution(key types.ExecutionKey) int32 {
	return s.shardController.GetShardIDForExecution(key)
}

func (s *Service) ResetExecution(ctx context.Context, key types.ExecutionKey, reason string, resetEventID int64) (string, error) {
	// TODO: Implement reset logic (replay history, branch execution, etc.)
	return "", errors.New("reset execution not implemented")
}
