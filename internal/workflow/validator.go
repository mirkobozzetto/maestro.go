package workflow

import (
	"fmt"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateDAG(workflow *Workflow) error {
	stepMap := make(map[string]*Step)
	dependencies := make(map[string][]string)

	for i := range workflow.Steps {
		if err := v.buildStepMap(&workflow.Steps[i], stepMap, dependencies); err != nil {
			return err
		}
	}

	if err := v.detectCycles(dependencies); err != nil {
		return fmt.Errorf("workflow contains cycles: %w", err)
	}

	return nil
}

func (v *Validator) buildStepMap(step *Step, stepMap map[string]*Step, deps map[string][]string) error {
	if len(step.Parallel) > 0 {
		for i := range step.Parallel {
			if err := v.buildStepMap(&step.Parallel[i], stepMap, deps); err != nil {
				return err
			}
		}
		return nil
	}

	if step.ID == "" {
		return fmt.Errorf("step must have an ID")
	}

	if _, exists := stepMap[step.ID]; exists {
		return fmt.Errorf("duplicate step ID: %s", step.ID)
	}

	stepMap[step.ID] = step

	if step.Output != "" {
		for key, value := range step.Input {
			if strVal, ok := value.(string); ok && IsTemplate(strVal) {
				referencedSteps := v.extractStepReferences(strVal)
				for _, ref := range referencedSteps {
					if ref != "input" {
						deps[step.ID] = append(deps[step.ID], ref)
					}
				}
				_ = key
			}
		}
	}

	return nil
}

func (v *Validator) extractStepReferences(template string) []string {
	var refs []string
	i := 0
	for i < len(template) {
		if i+3 < len(template) && template[i:i+3] == "{{ " {
			j := i + 3
			for j < len(template)-2 && template[j:j+2] != " }" {
				j++
			}
			if j < len(template)-2 {
				expr := template[i+3 : j]
				if len(expr) > 1 && expr[0] == '.' {
					parts := splitExpression(expr[1:])
					if len(parts) > 0 {
						refs = append(refs, parts[0])
					}
				}
				i = j + 2
			} else {
				i++
			}
		} else {
			i++
		}
	}
	return refs
}

func splitExpression(expr string) []string {
	var parts []string
	var current string
	for _, ch := range expr {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func (v *Validator) detectCycles(dependencies map[string][]string) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range dependencies {
		if !visited[node] {
			if v.hasCycleDFS(node, dependencies, visited, recStack) {
				return fmt.Errorf("cycle detected involving step: %s", node)
			}
		}
	}

	return nil
}

func (v *Validator) hasCycleDFS(node string, deps map[string][]string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range deps[node] {
		if !visited[neighbor] {
			if v.hasCycleDFS(neighbor, deps, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}