package orchestrator

import (
	"context"
	"fmt"

	"github.com/maestro/maestro.go/internal/workflow"
	"github.com/rs/zerolog"
)

type SagaCoordinator struct {
	executor *Executor
	logger   zerolog.Logger
}

func NewSagaCoordinator(executor *Executor, logger zerolog.Logger) *SagaCoordinator {
	return &SagaCoordinator{
		executor: executor,
		logger:   logger,
	}
}

func (s *SagaCoordinator) Compensate(
	ctx context.Context,
	execCtx *workflow.ExecutionContext,
	wf *workflow.Workflow,
) error {
	if len(execCtx.ExecutedSteps) == 0 {
		s.logger.Debug().Msg("No steps to compensate")
		return nil
	}

	workflowID := ctx.Value("workflow_id").(string)
	logger := s.logger.With().
		Str("workflow_id", workflowID).
		Int("steps_to_compensate", len(execCtx.ExecutedSteps)).
		Logger()

	logger.Info().Msg("Starting saga compensation")

	var compensationErrors []error

	for i := len(execCtx.ExecutedSteps) - 1; i >= 0; i-- {
		step := &execCtx.ExecutedSteps[i]

		if step.Compensation == nil {
			logger.Debug().
				Str("step_id", step.StepID).
				Msg("Step has no compensation, skipping")
			continue
		}

		if step.Compensated {
			logger.Debug().
				Str("step_id", step.StepID).
				Msg("Step already compensated, skipping")
			continue
		}

		err := s.executor.CompensateStep(ctx, step, execCtx, wf)
		if err != nil {
			logger.Error().
				Err(err).
				Str("step_id", step.StepID).
				Msg("Failed to compensate step")
			compensationErrors = append(compensationErrors, fmt.Errorf(
				"failed to compensate step %s: %w", step.StepID, err,
			))
			continue
		}

		logger.Info().
			Str("step_id", step.StepID).
			Msg("Step compensated successfully")
	}

	if len(compensationErrors) > 0 {
		return fmt.Errorf("compensation completed with %d errors: %v",
			len(compensationErrors), compensationErrors)
	}

	logger.Info().Msg("Saga compensation completed successfully")
	return nil
}

func (s *SagaCoordinator) RecordStep(
	execCtx *workflow.ExecutionContext,
	step *workflow.Step,
	result *workflow.StepResult,
) {
	if step.Compensate != nil {
		execCtx.ExecutedSteps = append(execCtx.ExecutedSteps, workflow.ExecutedStep{
			StepID:       step.ID,
			Output:       result.Output,
			Compensation: step.Compensate,
			Compensated:  false,
		})
	}

	if step.Output != "" && result != nil {
		execCtx.StepOutputs[step.Output] = result.Output
	}
}

type SagaState struct {
	WorkflowID    string
	ExecutedSteps []workflow.ExecutedStep
	Status        SagaStatus
}

type SagaStatus int

const (
	SagaStatusRunning SagaStatus = iota
	SagaStatusCompensating
	SagaStatusCompleted
	SagaStatusFailed
	SagaStatusCompensated
)

func (s SagaStatus) String() string {
	switch s {
	case SagaStatusRunning:
		return "running"
	case SagaStatusCompensating:
		return "compensating"
	case SagaStatusCompleted:
		return "completed"
	case SagaStatusFailed:
		return "failed"
	case SagaStatusCompensated:
		return "compensated"
	default:
		return "unknown"
	}
}