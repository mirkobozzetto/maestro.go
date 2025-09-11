package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/maestro/maestro.go/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (e *Executor) executeSingleStep(
	ctx context.Context,
	step *domain.Step,
	execCtx *domain.ExecutionContext,
	wf *domain.Workflow,
) (*domain.StepResult, error) {
	e.workerPool <- struct{}{}
	defer func() { <-e.workerPool }()

	workflowID := GetWorkflowID(ctx)
	logger := e.logger.With().
		Str("workflow_id", workflowID).
		Str("step_id", step.ID).
		Str("service", step.Service).
		Str("method", step.Method).
		Logger()

	logger.Info().Msg("Executing step")
	startTime := time.Now()

	service, exists := wf.Services[step.Service]
	if !exists {
		return nil, fmt.Errorf("service %s not found", step.Service)
	}

	resolvedInput, err := e.resolveStepInput(step, execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input: %w", err)
	}

	var result any
	var execErr error

	retryAttempts := 1
	if service.Retry != nil && service.Retry.Attempts > 1 {
		retryAttempts = service.Retry.Attempts
	}

	for attempt := 1; attempt <= retryAttempts; attempt++ {
		if attempt > 1 {
			backoffDuration := e.calculateBackoff(attempt-1, service.Retry)
			logger.Warn().
				Int("attempt", attempt).
				Dur("backoff", backoffDuration).
				Msg("Retrying step after backoff")
			time.Sleep(backoffDuration)
		}

		stepCtx := ctx
		if service.Timeout.Duration > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, service.Timeout.Duration)
			defer cancel()
		}

		result, execErr = e.client.InvokeMethod(
			stepCtx,
			step.Service,
			step.Method,
			resolvedInput,
			workflowID,
			step.ID,
		)

		if execErr == nil {
			break
		}

		if attempt < retryAttempts && isRetryableError(execErr) {
			logger.Warn().
				Err(execErr).
				Int("attempt", attempt).
				Msg("Step execution failed, will retry")
		} else if attempt < retryAttempts {
			logger.Error().
				Err(execErr).
				Msg("Step execution failed with non-retryable error")
			break
		}
	}

	if execErr != nil {
		logger.Error().
			Err(execErr).
			Dur("duration", time.Since(startTime)).
			Msg("Step execution failed after all retries")
		return nil, execErr
	}

	logger.Info().
		Dur("duration", time.Since(startTime)).
		Interface("output", result).
		Msg("Step executed successfully")

	return &domain.StepResult{
		StepID: step.ID,
		Output: result,
	}, nil
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
