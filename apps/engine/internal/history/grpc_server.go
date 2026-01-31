package history

import (
	historyv1 "github.com/linkflow/engine/api/gen/linkflow/history/v1"
)

type GRPCServer struct {
	historyv1.UnimplementedHistoryServiceServer
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}
