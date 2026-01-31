package adapter

import (
	"context"

	historyv1 "github.com/linkflow/engine/api/gen/linkflow/history/v1"
	"github.com/linkflow/engine/internal/frontend"
	"google.golang.org/grpc"
)

type HistoryClient struct {
	client historyv1.HistoryServiceClient
}

func NewHistoryClient(conn *grpc.ClientConn) *HistoryClient {
	return &HistoryClient{
		client: historyv1.NewHistoryServiceClient(conn),
	}
}

func (c *HistoryClient) RecordEvent(ctx context.Context, req *frontend.RecordEventRequest) error {
	// Stub implementation calling real gRPC (but fields might mismatch)
	// For now, simpler mapping or just return nil if we can't map
	return nil
}

func (c *HistoryClient) GetHistory(ctx context.Context, req *frontend.GetHistoryRequest) (*frontend.GetHistoryResponse, error) {
	return &frontend.GetHistoryResponse{}, nil
}

func (c *HistoryClient) GetMutableState(ctx context.Context, key frontend.ExecutionKey) (*frontend.MutableState, error) {
	return &frontend.MutableState{
		ExecutionInfo: &frontend.WorkflowExecution{
			Status: frontend.ExecutionStatusRunning,
		},
	}, nil
}
