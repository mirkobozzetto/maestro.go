package executor

import (
	"time"

	"github.com/maestro/maestro.go/internal/domain"
)

func (e *Executor) calculateBackoff(attempt int, retry *domain.RetryConfig) time.Duration {
	if retry == nil || retry.Backoff != "exponential" {
		return time.Second
	}

	baseDelay := time.Second
	maxDelay := 30 * time.Second
	delay := baseDelay * time.Duration(1<<uint(attempt))

	return min(delay, maxDelay)
}

