package domain

import (
	"time"
)

type Workflow struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Timeout  Duration          `yaml:"timeout"`
	Services map[string]Service `yaml:"services"`
	Steps    []Step            `yaml:"steps"`
	Output   map[string]string `yaml:"output"`
}

type Service struct {
	Type     string      `yaml:"type"`
	Endpoint string      `yaml:"endpoint"`
	Timeout  Duration    `yaml:"timeout"`
	Retry    *RetryConfig `yaml:"retry,omitempty"`
	Metadata map[string]string `yaml:"metadata,omitempty"`
}

type RetryConfig struct {
	Attempts int    `yaml:"attempts"`
	Backoff  string `yaml:"backoff"`
}

type Step struct {
	ID         string            `yaml:"id,omitempty"`
	Service    string            `yaml:"service,omitempty"`
	Method     string            `yaml:"method,omitempty"`
	Input      map[string]interface{} `yaml:"input,omitempty"`
	Output     string            `yaml:"output,omitempty"`
	When       string            `yaml:"when,omitempty"`
	Compensate *CompensateConfig `yaml:"compensate,omitempty"`
	Parallel   []Step            `yaml:"parallel,omitempty"`
}

type CompensateConfig struct {
	Method string                 `yaml:"method"`
	Input  map[string]interface{} `yaml:"input"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = duration
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

type ExecutionContext struct {
	WorkflowID string
	Input      map[string]interface{}
	Variables  map[string]interface{}
	StepOutputs map[string]interface{}
	ExecutedSteps []ExecutedStep
}

type ExecutedStep struct {
	StepID       string
	Output       interface{}
	Compensation *CompensateConfig
	Compensated  bool
}

type StepResult struct {
	StepID string
	Output interface{}
	Error  error
}

type WorkflowResult struct {
	WorkflowID string
	Status     WorkflowStatus
	Output     map[string]interface{}
	Error      error
	StartedAt  time.Time
	CompletedAt time.Time
}

type WorkflowStatus int

const (
	WorkflowStatusPending WorkflowStatus = iota
	WorkflowStatusRunning
	WorkflowStatusSuccess
	WorkflowStatusFailed
	WorkflowStatusCancelled
	WorkflowStatusCompensating
	WorkflowStatusCompensated
)

func (s WorkflowStatus) String() string {
	switch s {
	case WorkflowStatusPending:
		return "pending"
	case WorkflowStatusRunning:
		return "running"
	case WorkflowStatusSuccess:
		return "success"
	case WorkflowStatusFailed:
		return "failed"
	case WorkflowStatusCancelled:
		return "cancelled"
	case WorkflowStatusCompensating:
		return "compensating"
	case WorkflowStatusCompensated:
		return "compensated"
	default:
		return "unknown"
	}
}

func IsTemplate(s string) bool {
	return len(s) >= 4 && s[:2] == "{{" && s[len(s)-2:] == "}}"
}