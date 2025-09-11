package executor

import (
	"context"
	"fmt"
	"maps"

	"github.com/maestro/maestro.go/internal/domain"
)

func (e *Executor) CompensateStep(
	ctx context.Context,
	step *domain.ExecutedStep,
	execCtx *domain.ExecutionContext,
	wf *domain.Workflow,
) error {
	if step.Compensation == nil || step.Compensated {
		return nil
	}

	workflowID := GetWorkflowID(ctx)
	logger := e.logger.With().
		Str("workflow_id", workflowID).
		Str("step_id", step.StepID).
		Str("method", step.Compensation.Method).
		Logger()

	logger.Info().Msg("Compensating step")

	resolvedInput := make(map[string]any)
	templateData := make(map[string]any, len(execCtx.StepOutputs)+1)
	templateData["input"] = execCtx.Input
	maps.Copy(templateData, execCtx.StepOutputs)

	for key, value := range step.Compensation.Input {
		if strVal, ok := value.(string); ok && domain.IsTemplate(strVal) {
			resolved, err := e.resolveTemplate(strVal, templateData)
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
