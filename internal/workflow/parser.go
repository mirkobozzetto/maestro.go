package workflow

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Parser struct {
	templateEngine *template.Template
}

func NewParser() *Parser {
	return &Parser{
		templateEngine: template.New("workflow").Option("missingkey=error"),
	}
}

func (p *Parser) ParseFile(filename string) (*Workflow, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	return p.Parse(data)
}

func (p *Parser) Parse(data []byte) (*Workflow, error) {
	var workflow Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	if err := p.validateWorkflow(&workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	return &workflow, nil
}

func (p *Parser) validateWorkflow(w *Workflow) error {
	if w.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if w.Version == "" {
		return fmt.Errorf("workflow version is required")
	}

	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	for name, service := range w.Services {
		if err := p.validateService(name, &service); err != nil {
			return err
		}
	}

	for i, step := range w.Steps {
		if err := p.validateStep(&step, w.Services, i); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) validateService(name string, s *Service) error {
	if s.Type == "" {
		return fmt.Errorf("service %s: type is required", name)
	}

	if s.Endpoint == "" {
		return fmt.Errorf("service %s: endpoint is required", name)
	}

	if s.Type != "grpc" && s.Type != "http" {
		return fmt.Errorf("service %s: invalid type %s (must be 'grpc' or 'http')", name, s.Type)
	}

	return nil
}

func (p *Parser) validateStep(s *Step, services map[string]Service, index int) error {
	if len(s.Parallel) > 0 {
		for i, parallelStep := range s.Parallel {
			if err := p.validateStep(&parallelStep, services, i); err != nil {
				return fmt.Errorf("parallel step %d: %w", i, err)
			}
		}
		return nil
	}

	if s.ID == "" {
		s.ID = fmt.Sprintf("step_%d", index)
	}

	if s.Service == "" {
		return fmt.Errorf("step %s: service is required", s.ID)
	}

	if _, ok := services[s.Service]; !ok {
		return fmt.Errorf("step %s: unknown service %s", s.ID, s.Service)
	}

	if s.Method == "" {
		return fmt.Errorf("step %s: method is required", s.ID)
	}

	if s.Compensate != nil {
		if s.Compensate.Method == "" {
			return fmt.Errorf("step %s: compensation method is required", s.ID)
		}
	}

	return nil
}

func (p *Parser) ResolveTemplate(tmpl string, data interface{}) (string, error) {
	t, err := p.templateEngine.Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func (p *Parser) ResolveStepInput(step *Step, ctx *ExecutionContext) (map[string]interface{}, error) {
	resolvedInput := make(map[string]interface{})

	templateData := map[string]interface{}{
		"input": ctx.Input,
	}

	for stepID, output := range ctx.StepOutputs {
		templateData[stepID] = output
	}

	for key, value := range step.Input {
		switch v := value.(type) {
		case string:
			if IsTemplate(v) {
				resolved, err := p.ResolveTemplate(v, templateData)
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

func IsTemplate(s string) bool {
	return len(s) >= 4 && s[:2] == "{{" && s[len(s)-2:] == "}}"
}