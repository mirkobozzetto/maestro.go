package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maestro/maestro.go/internal/grpc"
	"github.com/maestro/maestro.go/internal/workflow"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Executor struct {
	registry   *grpc.ServiceRegistry
	client     *grpc.DynamicClient
	logger     zerolog.Logger
	workerPool chan struct{}
}

func NewExecutor(registry *grpc.ServiceRegistry, logger zerolog.Logger) *Executor {
	return &Executor{
		registry:   registry,
		client:     grpc.NewDynamicClient(registry, logger),
		logger:     logger,
		workerPool: make(chan struct{}, 10),
	}
}

func (e *Executor) ExecuteStep(
	ctx context.Context,
	step *workflow.Step,
	execCtx *workflow.ExecutionContext,
	wf *workflow.Workflow,
) (*workflow.StepResult, error) {
	if len(step.Parallel) > 0 {
		return e.executeParallelSteps(ctx, step.Parallel, execCtx, wf)
	}

	if step.When != "" {
		condition, err := e.evaluateCondition(step.When, execCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate condition: %w", err)
		}
		if !condition {
			e.logger.Debug().
				Str("step_id", step.ID).
				Str("condition", step.When).
				Msg("Skipping step due to condition")
			return &workflow.StepResult{
				StepID: step.ID,
				Output: nil,
			}, nil
		}
	}

	return e.executeSingleStep(ctx, step, execCtx, wf)
}

func (e *Executor) executeSingleStep(
	ctx context.Context,
	step *workflow.Step,
	execCtx *workflow.ExecutionContext,
	wf *workflow.Workflow,
) (*workflow.StepResult, error) {
	e.workerPool <- struct{}{}
	defer func() { <-e.workerPool }()

	workflowID := ctx.Value("workflow_id").(string)
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

	parser := workflow.NewParser()
	resolvedInput, err := parser.ResolveStepInput(step, execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input: %w", err)
	}

	var result interface{}
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

		if attempt < retryAttempts {
			logger.Warn().
				Err(execErr).
				Int("attempt", attempt).
				Msg("Step execution failed, will retry")
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

	return &workflow.StepResult{
		StepID: step.ID,
		Output: result,
	}, nil
}

func (e *Executor) executeParallelSteps(
	ctx context.Context,
	steps []workflow.Step,
	execCtx *workflow.ExecutionContext,
	wf *workflow.Workflow,
) (*workflow.StepResult, error) {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]*workflow.StepResult, len(steps))
	var mu sync.Mutex

	for i := range steps {
		idx := i
		step := steps[i]
		g.Go(func() error {
			result, err := e.ExecuteStep(ctx, &step, execCtx, wf)
			if err != nil {
				return err
			}

			mu.Lock()
			results[idx] = result
			if step.Output != "" && result != nil {
				execCtx.StepOutputs[step.Output] = result.Output
			}
			if step.Compensate != nil {
				execCtx.ExecutedSteps = append(execCtx.ExecutedSteps, workflow.ExecutedStep{
					StepID:       step.ID,
					Output:       result.Output,
					Compensation: step.Compensate,
				})
			}
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("parallel execution failed: %w", err)
	}

	combinedOutput := make(map[string]interface{})
	for i, result := range results {
		if result != nil && steps[i].Output != "" {
			combinedOutput[steps[i].Output] = result.Output
		}
	}

	return &workflow.StepResult{
		StepID: "parallel",
		Output: combinedOutput,
	}, nil
}

func (e *Executor) evaluateCondition(condition string, execCtx *workflow.ExecutionContext) (bool, error) {
	parser := workflow.NewParser()
	resolvedCondition, err := parser.ResolveTemplate(condition, map[string]interface{}{
		"input": execCtx.Input,
	})
	if err != nil {
		return false, err
	}

	for stepID, output := range execCtx.StepOutputs {
		if stepID == resolvedCondition {
			if b, ok := output.(bool); ok {
				return b, nil
			}
		}
	}

	return resolvedCondition == "true", nil
}

func (e *Executor) calculateBackoff(attempt int, retry *workflow.RetryConfig) time.Duration {
	if retry == nil || retry.Backoff != "exponential" {
		return time.Second
	}

	baseDelay := time.Second
	maxDelay := 30 * time.Second
	delay := baseDelay * time.Duration(1<<uint(attempt))

	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

func (e *Executor) CompensateStep(
	ctx context.Context,
	step *workflow.ExecutedStep,
	execCtx *workflow.ExecutionContext,
	wf *workflow.Workflow,
) error {
	if step.Compensation == nil || step.Compensated {
		return nil
	}

	workflowID := ctx.Value("workflow_id").(string)
	logger := e.logger.With().
		Str("workflow_id", workflowID).
		Str("step_id", step.StepID).
		Str("method", step.Compensation.Method).
		Logger()

	logger.Info().Msg("Compensating step")

	parser := workflow.NewParser()
	resolvedInput := make(map[string]interface{})
	templateData := map[string]interface{}{
		"input": execCtx.Input,
	}
	for stepID, output := range execCtx.StepOutputs {
		templateData[stepID] = output
	}

	for key, value := range step.Compensation.Input {
		if strVal, ok := value.(string); ok && workflow.IsTemplate(strVal) {
			resolved, err := parser.ResolveTemplate(strVal, templateData)
			if err != nil {
				return fmt.Errorf("failed to resolve compensation input: %w", err)
			}
			resolvedInput[key] = resolved
		} else {
			resolvedInput[key] = value
		}
	}

	_, err := e.client.InvokeMethod(
		ctx,
		step.StepID,
		step.Compensation.Method,
		resolvedInput,
		workflowID,
		step.StepID+"_compensate",
	)

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Compensation failed")
		return err
	}

	step.Compensated = true
	logger.Info().Msg("Step compensated successfully")
	return nil
}