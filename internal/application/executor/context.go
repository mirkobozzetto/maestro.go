package executor

import (
	"context"

	ctxkeys "github.com/maestro/maestro.go/internal/context"
)

func GetWorkflowID(ctx context.Context) string {
	if val := ctx.Value(ctxkeys.WorkflowID); val != nil {
		return val.(string)
	}
	return ""
}

func GetWorkflowName(ctx context.Context) string {
	if val := ctx.Value(ctxkeys.WorkflowName); val != nil {
		return val.(string)
	}
	return ""
}

func GetStepID(ctx context.Context) string {
	if val := ctx.Value(ctxkeys.StepID); val != nil {
		return val.(string)
	}
	return ""
}
