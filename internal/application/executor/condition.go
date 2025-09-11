package executor

import (
	"github.com/maestro/maestro.go/internal/domain"
)

func (e *Executor) evaluateCondition(condition string, execCtx *domain.ExecutionContext) (bool, error) {
	resolvedCondition, err := e.resolveTemplate(condition, map[string]any{
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
