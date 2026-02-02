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
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	// Encode Namespace in Token so worker can extract it (since PollTaskResponse lacks Namespace field)
	token := fmt.Sprintf("%s|%s", req.Namespace, taskID)
	task := &engine.Task{
		ID:               taskID,
		Token:            []byte(token),
		WorkflowID:       req.WorkflowExecution.GetWorkflowId(),
		RunID:            req.WorkflowExecution.GetRunId(),
		ScheduledTime:    req.ScheduleTime.AsTime(),
		TaskType:         int32(req.TaskType),
		ScheduledEventID: req.ScheduledEventId,
		ActivityID:       fmt.Sprintf("%d", req.ScheduledEventId), // Using event ID as activity ID for now if not provided
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
	resp := &matchingv1.PollTaskResponse{
		TaskToken: task.Token,
		WorkflowExecution: &commonv1.WorkflowExecution{
			WorkflowId: task.WorkflowID,
			RunId:      task.RunID,
		},
		Attempt:        task.Attempt,
		StartedEventId: 1, // Placeholder
	}

	if commonv1.TaskType(task.TaskType) == commonv1.TaskType_TASK_TYPE_WORKFLOW_TASK {
		resp.WorkflowTaskInfo = &matchingv1.WorkflowTaskInfo{
			ScheduledEventId: task.ScheduledEventID,
		}
	} else {
		resp.ActivityTaskInfo = &matchingv1.ActivityTaskInfo{
			ActivityId:       task.ActivityID,
			ActivityType:     task.ActivityType,
			ScheduledEventId: task.ScheduledEventID,
		}
		if len(task.Input) > 0 {
			resp.ActivityTaskInfo.Input = &commonv1.Payloads{
				Payloads: []*commonv1.Payload{{Data: task.Input}},
			}
		}
	}

	return resp, nil
}
