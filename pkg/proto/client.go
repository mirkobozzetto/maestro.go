package proto

import (
	"context"

	"google.golang.org/grpc"
)

type maestroServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMaestroServiceClient(cc grpc.ClientConnInterface) MaestroServiceClient {
	return &maestroServiceClient{cc}
}

func (c *maestroServiceClient) Execute(ctx context.Context, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error) {
	out := new(ServiceResponse)
	err := c.cc.Invoke(ctx, "/maestro.v1.MaestroService/Execute", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *maestroServiceClient) Compensate(ctx context.Context, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error) {
	out := new(ServiceResponse)
	err := c.cc.Invoke(ctx, "/maestro.v1.MaestroService/Compensate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *maestroServiceClient) HealthCheck(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HealthStatus, error) {
	out := new(HealthStatus)
	err := c.cc.Invoke(ctx, "/maestro.v1.MaestroService/HealthCheck", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
