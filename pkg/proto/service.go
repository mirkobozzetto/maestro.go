package proto

import (
	"context"

	"google.golang.org/grpc"
)

type MaestroServiceServer interface {
	Execute(context.Context, *ServiceRequest) (*ServiceResponse, error)
	Compensate(context.Context, *ServiceRequest) (*ServiceResponse, error)
	HealthCheck(context.Context, *Empty) (*HealthStatus, error)
}

type MaestroServiceClient interface {
	Execute(ctx context.Context, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error)
	Compensate(ctx context.Context, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error)
	HealthCheck(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HealthStatus, error)
}
