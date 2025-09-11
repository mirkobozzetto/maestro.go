package context

type Key string

const (
	WorkflowID   Key = "workflow_id"
	WorkflowName Key = "workflow_name"
	StepID       Key = "step_id"
)
