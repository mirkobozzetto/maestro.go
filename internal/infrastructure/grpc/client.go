package grpc

import (
	"context"
	"fmt"
	"time"

	ctxkeys "github.com/maestro/maestro.go/internal/context"
	adapters "github.com/maestro/maestro.go/internal/infrastructure/http"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/maestro/maestro.go/pkg/proto"
)

type DynamicClient struct {
	registry *ServiceRegistry
	logger   zerolog.Logger
}

func NewDynamicClient(registry *ServiceRegistry, logger zerolog.Logger) *DynamicClient {
	return &DynamicClient{
		registry: registry,
		logger:   logger,
	}
}

func (c *DynamicClient) InvokeMethod(
	ctx context.Context,
	serviceName string,
	method string,
	input map[string]interface{},
	workflowID string,
	stepID string,
) (interface{}, error) {
	service, err := c.registry.GetService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	if service.Config.Type == "http" {
		return c.invokeHTTP(ctx, service, method, input, workflowID, stepID)
	}

	return c.invokeGRPC(ctx, serviceName, service, method, input, workflowID, stepID)
}

func (c *DynamicClient) invokeGRPC(
	ctx context.Context,
	serviceName string,
	_ *ServiceEntry,
	method string,
	input map[string]interface{},
	workflowID string,
	stepID string,
) (interface{}, error) {
	conn, err := c.registry.GetConnection(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	cb, err := c.registry.GetCircuitBreaker(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get circuit breaker: %w", err)
	}

	payload, err := structpb.NewStruct(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create struct payload: %w", err)
	}

	payloadAny, err := anypb.New(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create any payload: %w", err)
	}

	req := &pb.ServiceRequest{
		Method:        method,
		Payload:       payloadAny,
		Headers:       make(map[string]string),
		CorrelationId: fmt.Sprintf("%s:%s", workflowID, stepID),
		WorkflowId:    workflowID,
		StepId:        stepID,
	}

	md := metadata.New(map[string]string{
		"workflow-id":    workflowID,
		"step-id":        stepID,
		"correlation-id": req.CorrelationId,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	var result interface{}
	operation := func() (interface{}, error) {
		client := pb.NewMaestroServiceClient(conn)
		resp, err := client.Execute(ctx, req)
		if err != nil {
			return nil, err
		}

		if !resp.Success {
			return nil, fmt.Errorf("service returned error: %s", resp.Error)
		}

		if resp.Data != nil {
			var structData structpb.Struct
			if err := resp.Data.UnmarshalTo(&structData); err != nil {
				result = resp.Data.String()
			} else {
				result = structData.AsMap()
			}
		}

		return result, nil
	}

	resultInterface, err := cb.Execute(operation)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
				c.registry.UpdateHealth(serviceName, false)
			}
		}
		return nil, fmt.Errorf("gRPC invocation failed: %w", err)
	}

	if resultInterface != nil {
		return resultInterface, nil
	}

	return result, nil
}

func (c *DynamicClient) invokeHTTP(
	_ context.Context,
	service *ServiceEntry,
	method string,
	input map[string]interface{},
	workflowID string,
	stepID string,
) (interface{}, error) {
	adapter := adapters.NewHTTPAdapter()
	result, err := adapter.InvokeHTTP(service.Config.Endpoint, method, input)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("service_type", "http").
			Str("method", method).
			Str("workflow_id", workflowID).
			Str("step_id", stepID).
			Msg("HTTP invocation failed")
		return nil, err
	}

	c.logger.Info().
		Str("service_type", "http").
		Str("method", method).
		Str("workflow_id", workflowID).
		Str("step_id", stepID).
		Interface("result", result).
		Msg("HTTP invocation successful")

	return result, nil
}

type InvocationOptions struct {
	Timeout        time.Duration
	RetryAttempts  int
	RetryBackoff   time.Duration
	CircuitBreaker bool
	TraceEnabled   bool
}

func (c *DynamicClient) InvokeWithOptions(
	ctx context.Context,
	serviceName string,
	method string,
	input map[string]any,
	opts InvocationOptions,
) (any, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	workflowID := ""
	if val := ctx.Value(ctxkeys.WorkflowID); val != nil {
		workflowID = val.(string)
	}

	stepID := ""
	if val := ctx.Value(ctxkeys.StepID); val != nil {
		stepID = val.(string)
	}

	var result any
	var err error

	for attempt := 0; attempt <= opts.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(opts.RetryBackoff * time.Duration(attempt))
		}

		result, err = c.InvokeMethod(ctx, serviceName, method, input, workflowID, stepID)
		if err == nil {
			return result, nil
		}

		if !isRetryableError(err) {
			break
		}
	}

	return nil, err
}

func isRetryableError(err error) bool {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
			return true
		}
	}
	return false
}
