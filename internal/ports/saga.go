package ports

import (
	"context"
	"github.com/maestro/maestro.go/internal/domain"
)

type SagaCoordinator interface {
	Compensate(ctx context.Context, execCtx *domain.ExecutionContext, workflow *domain.Workflow) error
	RecordStep(execCtx *domain.ExecutionContext, step *domain.Step, result *domain.StepResult)
}