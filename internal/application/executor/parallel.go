package executor

import (
	"context"
	"fmt"
	"sync"

	"github.com/maestro/maestro.go/internal/domain"
	"golang.org/x/sync/errgroup"
)

func (e *Executor) executeParallelSteps(
	ctx context.Context,
	steps []domain.Step,
	execCtx *domain.ExecutionContext,
	wf *domain.Workflow,
) (*domain.StepResult, error) {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]*domain.StepResult, len(steps))
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
				execCtx.ExecutedSteps = append(execCtx.ExecutedSteps, domain.ExecutedStep{
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

	combinedOutput := make(map[string]any)
	for i, result := range results {
		if result != nil && steps[i].Output != "" {
			combinedOutput[steps[i].Output] = result.Output
		}
	}

	return &domain.StepResult{
		StepID: "parallel",
		Output: combinedOutput,
	}, nil
}
