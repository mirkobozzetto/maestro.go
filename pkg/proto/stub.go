package proto

import (
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MaestroServiceClient interface {
	Execute(ctx interface{}, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error)
	Compensate(ctx interface{}, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error)
	HealthCheck(ctx interface{}, in *Empty, opts ...grpc.CallOption) (*HealthStatus, error)
}

type maestroServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMaestroServiceClient(cc grpc.ClientConnInterface) MaestroServiceClient {
	return &maestroServiceClient{cc}
}

func (c *maestroServiceClient) Execute(ctx interface{}, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error) {
	return &ServiceResponse{Success: true}, nil
}

func (c *maestroServiceClient) Compensate(ctx interface{}, in *ServiceRequest, opts ...grpc.CallOption) (*ServiceResponse, error) {
	return &ServiceResponse{Success: true}, nil
}

func (c *maestroServiceClient) HealthCheck(ctx interface{}, in *Empty, opts ...grpc.CallOption) (*HealthStatus, error) {
	return &HealthStatus{Healthy: true}, nil
}

type ServiceRequest struct {
	Method        string
	Payload       *anypb.Any
	Headers       map[string]string
	CorrelationId string
	WorkflowId    string
	StepId        string
}

type ServiceResponse struct {
	Success  bool
	Data     *anypb.Any
	Error    string
	Metadata map[string]string
}

type Empty struct{}

type HealthStatus struct {
	Healthy   bool
	Message   string
	CheckedAt *timestamppb.Timestamp
}

type ExecuteRequest struct {
	WorkflowName   string
	Input          *structpb.Struct
	IdempotencyKey string
	Metadata       map[string]string
}

type ExecuteResponse struct {
	WorkflowId   string
	Status       WorkflowStatus
	Output       *structpb.Struct
	Error        string
	StartedAt    *timestamppb.Timestamp
	CompletedAt  *timestamppb.Timestamp
}

type WorkflowStatus int32

const (
	WorkflowStatus_WORKFLOW_STATUS_UNKNOWN      WorkflowStatus = 0
	WorkflowStatus_WORKFLOW_STATUS_PENDING      WorkflowStatus = 1
	WorkflowStatus_WORKFLOW_STATUS_RUNNING      WorkflowStatus = 2
	WorkflowStatus_WORKFLOW_STATUS_SUCCESS      WorkflowStatus = 3
	WorkflowStatus_WORKFLOW_STATUS_FAILED       WorkflowStatus = 4
	WorkflowStatus_WORKFLOW_STATUS_CANCELLED    WorkflowStatus = 5
	WorkflowStatus_WORKFLOW_STATUS_COMPENSATING WorkflowStatus = 6
	WorkflowStatus_WORKFLOW_STATUS_COMPENSATED  WorkflowStatus = 7
)