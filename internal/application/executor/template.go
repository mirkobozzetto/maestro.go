package executor

import (
	"bytes"
	"fmt"
	"maps"
	"text/template"

	"github.com/maestro/maestro.go/internal/domain"
)

func (e *Executor) resolveTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("executor").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func (e *Executor) resolveStepInput(step *domain.Step, ctx *domain.ExecutionContext) (map[string]any, error) {
	resolvedInput := make(map[string]any)

	templateData := make(map[string]any, len(ctx.StepOutputs)+1)
	templateData["input"] = ctx.Input
	maps.Copy(templateData, ctx.StepOutputs)

	for key, value := range step.Input {
		switch v := value.(type) {
		case string:
			if domain.IsTemplate(v) {
				resolved, err := e.resolveTemplate(v, templateData)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve template for key %s: %w", key, err)
				}
				resolvedInput[key] = resolved
			} else {
				resolvedInput[key] = v
			}
		default:
			resolvedInput[key] = value
		}
	}

	return resolvedInput, nil
}
