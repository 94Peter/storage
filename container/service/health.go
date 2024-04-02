package service

import (
	"context"

	"github.com/94peter/microservice/grpc_tool/health/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewHealthService() pb.HealthServer {
	return &healthServer{}
}

type healthServer struct {
	pb.UnimplementedHealthServer
}

func (s *healthServer) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}

func (s *healthServer) Watch(req *pb.HealthCheckRequest, stream pb.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "method Watch not implemented")
}
