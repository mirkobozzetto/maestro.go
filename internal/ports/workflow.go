package ports

import (
	"context"
	"github.com/maestro/maestro.go/internal/domain"
)

type WorkflowParser interface {
	ParseFile(filename string) (*domain.Workflow, error)
	Parse(data []byte) (*domain.Workflow, error)
}

type WorkflowExecutor interface {
	Execute(ctx context.Context, workflow *domain.Workflow, input map[string]interface{}) (*domain.WorkflowResult, error)
}

type ServiceInvoker interface {
	InvokeMethod(ctx context.Context, service, method string, input map[string]interface{}) (interface{}, error)
}