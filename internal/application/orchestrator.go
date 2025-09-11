package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	workflow "github.com/maestro/maestro.go/internal/domain"
	"github.com/maestro/maestro.go/internal/infrastructure/grpc"
	"github.com/rs/zerolog"
)

type Orchestrator struct {
	mu               sync.RWMutex
	workflows        map[string]*workflow.Workflow
	parser           *Parser
	executor         *Executor
	sagaCoordinator  *SagaCoordinator
	registry         *grpc.ServiceRegistry
	logger           zerolog.Logger
	runningWorkflows sync.Map
}

func New(logger zerolog.Logger) *Orchestrator {
	registry := grpc.NewServiceRegistry()
	executor := NewExecutor(registry, logger)
	sagaCoordinator := NewSagaCoordinator(executor, logger)

	return &Orchestrator{
		workflows:       make(map[string]*workflow.Workflow),
		parser:          NewParser(),
		executor:        executor,
		sagaCoordinator: sagaCoordinator,
		registry:        registry,
		logger:          logger,
	}
}

func (o *Orchestrator) LoadWorkflow(filename string) error {
	wf, err := o.parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	o.workflows[wf.Name] = wf

	for name, service := range wf.Services {
		if err := o.registry.RegisterService(name, &service); err != nil {
			return fmt.Errorf("failed to register service %s: %w", name, err)
		}
	}

	o.logger.Info().
		Str("workflow", wf.Name).
		Str("version", wf.Version).
		Int("steps", len(wf.Steps)).
		Msg("Workflow loaded successfully")

	return nil
}

func (o *Orchestrator) ExecuteWorkflow(
	ctx context.Context,
	workflowName string,
	input map[string]interface{},
) (*workflow.WorkflowResult, error) {
	o.mu.RLock()
	wf, exists := o.workflows[workflowName]
	o.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowName)
	}

	workflowID := uuid.New().String()
	logger := o.logger.With().
		Str("workflow_id", workflowID).
		Str("workflow_name", workflowName).
		Logger()

	logger.Info().
		Interface("input", input).
		Msg("Starting workflow execution")

	execCtx := &workflow.ExecutionContext{
		WorkflowID:    workflowID,
		Input:         input,
		Variables:     make(map[string]interface{}),
		StepOutputs:   make(map[string]interface{}),
		ExecutedSteps: []workflow.ExecutedStep{},
	}

	if wf.Timeout.Duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, wf.Timeout.Duration)
		defer cancel()
	}

	ctx = context.WithValue(ctx, "workflow_id", workflowID)
	ctx = context.WithValue(ctx, "workflow_name", workflowName)

	startedAt := time.Now()
	result := &workflow.WorkflowResult{
		WorkflowID: workflowID,
		Status:     workflow.WorkflowStatusRunning,
		StartedAt:  startedAt,
	}

	o.runningWorkflows.Store(workflowID, result)
	defer o.runningWorkflows.Delete(workflowID)

	for _, step := range wf.Steps {
		select {
		case <-ctx.Done():
			result.Status = workflow.WorkflowStatusCancelled
			result.Error = ctx.Err()
			result.CompletedAt = time.Now()
			return result, ctx.Err()
		default:
		}

		stepResult, err := o.executor.ExecuteStep(ctx, &step, execCtx, wf)
		if err != nil {
			logger.Error().
				Err(err).
				Str("step_id", step.ID).
				Msg("Step execution failed")

			compensationErr := o.sagaCoordinator.Compensate(ctx, execCtx, wf)
			if compensationErr != nil {
				logger.Error().
					Err(compensationErr).
					Msg("Compensation failed")
				result.Status = workflow.WorkflowStatusFailed
			} else {
				result.Status = workflow.WorkflowStatusCompensated
			}

			result.Error = err
			result.CompletedAt = time.Now()
			return result, err
		}

		if stepResult != nil {
			if step.Output != "" {
				execCtx.StepOutputs[step.Output] = stepResult.Output
			}

			if step.Compensate != nil {
				execCtx.ExecutedSteps = append(execCtx.ExecutedSteps, workflow.ExecutedStep{
					StepID:       step.ID,
					Output:       stepResult.Output,
					Compensation: step.Compensate,
				})
			}
		}
	}

	resultOutput := make(map[string]interface{})
	for key, tmpl := range wf.Output {
		value, err := o.parser.ResolveTemplate(tmpl, map[string]interface{}{
			"input": execCtx.Input,
		})
		if err != nil {
			logger.Warn().
				Err(err).
				Str("key", key).
				Msg("Failed to resolve output template")
			continue
		}
		resultOutput[key] = value
	}

	for stepName, output := range execCtx.StepOutputs {
		if outputMap, ok := resultOutput[stepName]; !ok {
			resultOutput[stepName] = output
		} else {
			_ = outputMap
		}
	}

	result.Status = workflow.WorkflowStatusSuccess
	result.Output = resultOutput
	result.CompletedAt = time.Now()

	logger.Info().
		Str("status", result.Status.String()).
		Dur("duration", result.CompletedAt.Sub(result.StartedAt)).
		Interface("output", result.Output).
		Msg("Workflow execution completed")

	return result, nil
}

func (o *Orchestrator) GetWorkflowStatus(workflowID string) (*workflow.WorkflowResult, bool) {
	if result, ok := o.runningWorkflows.Load(workflowID); ok {
		return result.(*workflow.WorkflowResult), true
	}
	return nil, false
}

func (o *Orchestrator) CancelWorkflow(workflowID string) error {
	if result, ok := o.runningWorkflows.Load(workflowID); ok {
		wfResult := result.(*workflow.WorkflowResult)
		wfResult.Status = workflow.WorkflowStatusCancelled
		wfResult.CompletedAt = time.Now()
		return nil
	}
	return fmt.Errorf("workflow %s not found", workflowID)
}

func (o *Orchestrator) ListWorkflows() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	names := make([]string, 0, len(o.workflows))
	for name := range o.workflows {
		names = append(names, name)
	}
	return names
}
