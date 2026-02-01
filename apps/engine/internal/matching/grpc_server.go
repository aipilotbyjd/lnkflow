package matching

import (
	"context"
	"fmt"
	"time"

	commonv1 "github.com/linkflow/engine/api/gen/linkflow/common/v1"
	matchingv1 "github.com/linkflow/engine/api/gen/linkflow/matching/v1"
	"github.com/linkflow/engine/internal/matching/engine"
)

type GRPCServer struct {
	matchingv1.UnimplementedMatchingServiceServer
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) AddTask(ctx context.Context, req *matchingv1.AddTaskRequest) (*matchingv1.AddTaskResponse, error) {
	// Map proto request to internal engine.Task
	task := &engine.Task{
		ID:            fmt.Sprintf("task-%d", time.Now().UnixNano()), // Simple ID generation
		WorkflowID:    req.WorkflowExecution.GetWorkflowId(),
		RunID:         req.WorkflowExecution.GetRunId(),
		ScheduledTime: req.ScheduleTime.AsTime(),
		// We map what we can. Internal Task struct seems simplified.
	}

	queueName := req.TaskQueue.GetName()
	if queueName == "" {
		queueName = "default"
	}

	err := s.service.AddTask(ctx, queueName, task)
	if err != nil {
		return nil, err
	}

	return &matchingv1.AddTaskResponse{}, nil
}

func (s *GRPCServer) PollTask(ctx context.Context, req *matchingv1.PollTaskRequest) (*matchingv1.PollTaskResponse, error) {
	queueName := req.TaskQueue.GetName()
	if queueName == "" {
		queueName = "default"
	}

	// Auto-create task queue if it doesn't exist (workers poll before tasks arrive)
	s.service.GetOrCreateTaskQueue(queueName, engine.TaskQueueKindNormal)

	task, err := s.service.PollTask(ctx, queueName, req.Identity)
	if err != nil {
		return nil, err
	}

	// Map internal engine.Task to proto PollTaskResponse
	return &matchingv1.PollTaskResponse{
		TaskToken: task.Token,
		WorkflowExecution: &commonv1.WorkflowExecution{
			WorkflowId: task.WorkflowID,
			RunId:      task.RunID,
		},
		Attempt:        task.Attempt,
		StartedEventId: 1, // Placeholder
		// Logic to map other fields would go here
	}, nil
}
