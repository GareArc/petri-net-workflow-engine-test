package dsl

import (
	"fmt"
	"os"
	"petri-net-mvp/core/workflow"

	"gopkg.in/yaml.v3"
)

// WorkflowYAML represents the YAML structure
type WorkflowYAML struct {
	Workflow struct {
		Name      string         `yaml:"name"`
		Resources []ResourceYAML `yaml:"resources,omitempty"`
		Contexts  []ContextYAML  `yaml:"contexts,omitempty"`
		Channels  []ChannelYAML  `yaml:"channels,omitempty"`
		Tasks     []TaskYAML     `yaml:"tasks"`
		Gateways  []GatewayYAML  `yaml:"gateways,omitempty"`
	} `yaml:"workflow"`
}

type ResourceYAML struct {
	ID       string `yaml:"id"`
	Type     string `yaml:"type"`
	Capacity int    `yaml:"capacity"`
}

type ContextYAML struct {
	ID       string `yaml:"id"`
	Capacity int    `yaml:"capacity,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

type ChannelYAML struct {
	ID       string `yaml:"id"`
	Capacity int    `yaml:"capacity"`
	Type     string `yaml:"type,omitempty"`
}

type TaskYAML struct {
	ID       string                 `yaml:"id"`
	Type     string                 `yaml:"type"`
	Input    string                 `yaml:"input,omitempty"`
	Output   string                 `yaml:"output,omitempty"`
	Inputs   []string               `yaml:"inputs,omitempty"`
	Outputs  []string               `yaml:"outputs,omitempty"`
	Requires map[string]int         `yaml:"requires,omitempty"`
	Parallel bool                   `yaml:"parallel,omitempty"`
	Context  string                 `yaml:"context,omitempty"`
	Config   map[string]interface{} `yaml:"config,omitempty"`

	// Task-specific fields
	Model  string `yaml:"model,omitempty"`
	Prompt string `yaml:"prompt,omitempty"`
	URL    string `yaml:"url,omitempty"`
	Script string `yaml:"script,omitempty"`
	Source string `yaml:"source,omitempty"`
	Dest   string `yaml:"destination,omitempty"`
}

type GatewayYAML struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Inputs  []string `yaml:"inputs,omitempty"`
	Outputs []string `yaml:"outputs,omitempty"`
	WaitFor []string `yaml:"wait_for,omitempty"`
}

// Parser parses YAML workflow definitions
type Parser struct{}

// NewParser creates a new DSL parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a YAML workflow file
func (p *Parser) ParseFile(filename string) (*workflow.Workflow, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(data)
}

// Parse parses YAML workflow data
func (p *Parser) Parse(data []byte) (*workflow.Workflow, error) {
	var wfYAML WorkflowYAML
	if err := yaml.Unmarshal(data, &wfYAML); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	wf := &workflow.Workflow{
		Name:      wfYAML.Workflow.Name,
		Resources: make([]workflow.Resource, len(wfYAML.Workflow.Resources)),
		Contexts:  make([]workflow.Context, len(wfYAML.Workflow.Contexts)),
		Channels:  make([]workflow.Channel, len(wfYAML.Workflow.Channels)),
		Tasks:     make([]workflow.Task, len(wfYAML.Workflow.Tasks)),
		Gateways:  make([]workflow.Gateway, len(wfYAML.Workflow.Gateways)),
	}

	// Convert resources
	for i, r := range wfYAML.Workflow.Resources {
		wf.Resources[i] = workflow.Resource{
			ID:       r.ID,
			Type:     r.Type,
			Capacity: r.Capacity,
		}
	}

	// Convert contexts
	for i, c := range wfYAML.Workflow.Contexts {
		capacity := c.Capacity
		if capacity == 0 {
			capacity = 1
		}
		wf.Contexts[i] = workflow.Context{
			ID:       c.ID,
			Type:     c.Type,
			Capacity: capacity,
		}
	}

	// Convert channels
	for i, c := range wfYAML.Workflow.Channels {
		wf.Channels[i] = workflow.Channel{
			ID:       c.ID,
			Capacity: c.Capacity,
			Type:     c.Type,
		}
	}

	// Convert tasks
	for i, t := range wfYAML.Workflow.Tasks {
		task := workflow.Task{
			ID:       t.ID,
			Type:     t.Type,
			Input:    t.Input,
			Output:   t.Output,
			Inputs:   t.Inputs,
			Outputs:  t.Outputs,
			Requires: t.Requires,
			Parallel: t.Parallel,
			Context:  t.Context,
			Config:   make(map[string]interface{}),
		}

		// Populate config from task-specific fields
		if t.Model != "" {
			task.Config["model"] = t.Model
		}
		if t.Prompt != "" {
			task.Config["prompt"] = t.Prompt
		}
		if t.URL != "" {
			task.Config["url"] = t.URL
		}
		if t.Script != "" {
			task.Config["script"] = t.Script
		}
		if t.Source != "" {
			task.Config["source"] = t.Source
		}
		if t.Dest != "" {
			task.Config["destination"] = t.Dest
		}

		// Merge additional config
		for k, v := range t.Config {
			task.Config[k] = v
		}

		wf.Tasks[i] = task
	}

	// Convert gateways
	for i, g := range wfYAML.Workflow.Gateways {
		wf.Gateways[i] = workflow.Gateway{
			ID:      g.ID,
			Type:    g.Type,
			Inputs:  g.Inputs,
			Outputs: g.Outputs,
			WaitFor: g.WaitFor,
		}
	}

	if err := workflow.Validate(wf); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	return wf, nil
}
