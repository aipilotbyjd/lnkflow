package executor

import "encoding/json"

type JobPayload struct {
	JobID       string                 `json:"job_id"`
	Workflow    WorkflowDefinition     `json:"workflow"`
	TriggerData map[string]interface{} `json:"trigger_data"`
	Credentials map[string]interface{} `json:"credentials"`
	Variables   map[string]interface{} `json:"variables"`
}

type WorkflowDefinition struct {
	Nodes    []Node                 `json:"nodes"`
	Edges    []Edge                 `json:"edges"`
	Settings map[string]interface{} `json:"settings"`
}

type Node struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Position Position        `json:"position"`
	Data     json.RawMessage `json:"data"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}
