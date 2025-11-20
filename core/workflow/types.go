package workflow

import "context"

// Workflow represents a high-level workflow definition
type Workflow struct {
	Name      string
	Resources []Resource
	Contexts  []Context
	Channels  []Channel
	Tasks     []Task
	Gateways  []Gateway
}

// Resource represents a shared resource with capacity
type Resource struct {
	ID       string
	Type     string // "semaphore", "pool", "quota"
	Capacity int    // -1 = unlimited
}

// Context represents shared workflow state held in a dedicated place
type Context struct {
	ID       string
	Type     string // "context"
	Capacity int    // default 1
}

// Channel represents a data flow channel
type Channel struct {
	ID       string
	Capacity int    // -1 = unlimited
	Type     string // "fifo", "lifo", "priority"
}

// Task represents a unit of work
type Task struct {
	ID       string
	Type     string
	Input    string         // Channel ID
	Output   string         // Channel ID
	Inputs   []string       // Multiple inputs
	Outputs  []string       // Multiple outputs
	Requires map[string]int // Resource requirements: resource_id -> amount
	Parallel bool           // Auto-spawn workers
	Context  string         // Optional context place ID
	Action   TaskAction
	Config   map[string]interface{}
}

// TaskAction is the function executed by a task
type TaskAction func(ctx context.Context, input interface{}) (interface{}, error)

// Gateway represents control flow (barrier, split, merge)
type Gateway struct {
	ID      string
	Type    string   // "barrier", "split", "merge"
	Inputs  []string // Task IDs to wait for
	Outputs []string // Task IDs to trigger
	WaitFor []string // Alias for Inputs
}
