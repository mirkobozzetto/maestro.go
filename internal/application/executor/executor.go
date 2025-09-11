package executor

import (
	"context"

	"github.com/maestro/maestro.go/internal/domain"
	"github.com/maestro/maestro.go/internal/infrastructure/grpc"
	"github.com/rs/zerolog"
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
	step *domain.Step,
	execCtx *domain.ExecutionContext,
	wf *domain.Workflow,
) (*domain.StepResult, error) {
	if len(step.Parallel) > 0 {
		return e.executeParallelSteps(ctx, step.Parallel, execCtx, wf)
	}

	if step.When != "" {
		condition, err := e.evaluateCondition(step.When, execCtx)
		if err != nil {
			return nil, err
		}
		if !condition {
			e.logger.Debug().
				Str("step_id", step.ID).
				Str("condition", step.When).
				Msg("Skipping step due to condition")
			return &domain.StepResult{
				StepID: step.ID,
				Output: nil,
			}, nil
		}
	}

	return e.executeSingleStep(ctx, step, execCtx, wf)
}
