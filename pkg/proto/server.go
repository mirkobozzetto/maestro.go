package proto

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UnimplementedMaestroServiceServer struct{}

func (UnimplementedMaestroServiceServer) Execute(context.Context, *ServiceRequest) (*ServiceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Execute not implemented")
}

func (UnimplementedMaestroServiceServer) Compensate(context.Context, *ServiceRequest) (*ServiceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Compensate not implemented")
}

func (UnimplementedMaestroServiceServer) HealthCheck(context.Context, *Empty) (*HealthStatus, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HealthCheck not implemented")
}

func RegisterMaestroServiceServer(s *grpc.Server, srv MaestroServiceServer) {
	s.RegisterService(&MaestroServiceDesc, srv)
}
