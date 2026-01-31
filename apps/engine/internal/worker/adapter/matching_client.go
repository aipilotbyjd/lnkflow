package adapter

import (
	"context"

	commonv1 "github.com/linkflow/engine/api/gen/linkflow/common/v1"
	matchingv1 "github.com/linkflow/engine/api/gen/linkflow/matching/v1"
	"github.com/linkflow/engine/internal/worker/poller"
	"google.golang.org/grpc"
)

type MatchingClient struct {
	client matchingv1.MatchingServiceClient
}

func NewMatchingClient(conn *grpc.ClientConn) *MatchingClient {
	return &MatchingClient{
		client: matchingv1.NewMatchingServiceClient(conn),
	}
}

func (c *MatchingClient) PollTask(ctx context.Context, taskQueue string, identity string) (*poller.Task, error) {
	req := &matchingv1.PollTaskRequest{
		Namespace: "default",
		TaskQueue: &matchingv1.TaskQueue{
			Name: taskQueue,
			Kind: commonv1.TaskQueueKind_TASK_QUEUE_KIND_NORMAL,
		},
		Identity: identity,
	}

	resp, err := c.client.PollTask(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.TaskToken == nil {
		return nil, nil
	}

	var task *poller.Task

	if resp.ActivityTaskInfo != nil {
		task = &poller.Task{
			TaskID:     resp.ActivityTaskInfo.ActivityId,
			WorkflowID: resp.WorkflowExecution.GetWorkflowId(),
			RunID:      resp.WorkflowExecution.GetRunId(),
			NodeType:   resp.ActivityTaskInfo.ActivityType,
			Attempt:    resp.Attempt,
			TimeoutSec: 60, // Default timeout
		}

		if resp.ActivityTaskInfo.Input != nil && len(resp.ActivityTaskInfo.Input.Payloads) > 0 {
			task.Input = resp.ActivityTaskInfo.Input.Payloads[0].Data
		}
	} else if resp.WorkflowTaskInfo != nil {
		// Placeholder for Workflow Task
		// Not implemented yet for worker poller which expects Activity
		return nil, nil
	} else {
		return nil, nil
	}

	return task, nil
}
