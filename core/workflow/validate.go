package workflow

import "fmt"

// Validate ensures workflow definitions are internally consistent before compilation.
func Validate(wf *Workflow) error {
	resourceIDs := make(map[string]struct{})
	contextIDs := make(map[string]struct{})
	channelIDs := make(map[string]struct{})
	taskIDs := make(map[string]struct{})
	gatewayIDs := make(map[string]struct{})

	for _, r := range wf.Resources {
		if r.ID == "" {
			return fmt.Errorf("resource id cannot be empty")
		}
		if _, exists := resourceIDs[r.ID]; exists {
			return fmt.Errorf("duplicate resource id: %s", r.ID)
		}
		resourceIDs[r.ID] = struct{}{}
	}

	for _, c := range wf.Channels {
		if c.ID == "" {
			return fmt.Errorf("channel id cannot be empty")
		}
		if _, exists := channelIDs[c.ID]; exists {
			return fmt.Errorf("duplicate channel id: %s", c.ID)
		}
		channelIDs[c.ID] = struct{}{}
	}

	for _, c := range wf.Contexts {
		if c.ID == "" {
			return fmt.Errorf("context id cannot be empty")
		}
		if _, exists := contextIDs[c.ID]; exists {
			return fmt.Errorf("duplicate context id: %s", c.ID)
		}
		if _, conflict := resourceIDs[c.ID]; conflict {
			return fmt.Errorf("context id %s conflicts with resource id", c.ID)
		}
		if _, conflict := channelIDs[c.ID]; conflict {
			return fmt.Errorf("context id %s conflicts with channel id", c.ID)
		}
		contextIDs[c.ID] = struct{}{}
	}

	for _, t := range wf.Tasks {
		if t.ID == "" {
			return fmt.Errorf("task id cannot be empty")
		}
		if _, exists := taskIDs[t.ID]; exists {
			return fmt.Errorf("duplicate task id: %s", t.ID)
		}
		taskIDs[t.ID] = struct{}{}

		if t.Input != "" {
			if _, ok := channelIDs[t.Input]; !ok {
				return fmt.Errorf("task %s references missing input channel %s", t.ID, t.Input)
			}
		}
		for _, in := range t.Inputs {
			if _, ok := channelIDs[in]; !ok {
				return fmt.Errorf("task %s references missing input channel %s", t.ID, in)
			}
		}
		if t.Output != "" {
			if _, ok := channelIDs[t.Output]; !ok {
				return fmt.Errorf("task %s references missing output channel %s", t.ID, t.Output)
			}
		}
		for _, out := range t.Outputs {
			if _, ok := channelIDs[out]; !ok {
				return fmt.Errorf("task %s references missing output channel %s", t.ID, out)
			}
		}
		for resID := range t.Requires {
			if _, ok := resourceIDs[resID]; !ok {
				return fmt.Errorf("task %s requires missing resource %s", t.ID, resID)
			}
		}
		if t.Context != "" {
			if _, ok := contextIDs[t.Context]; !ok {
				return fmt.Errorf("task %s references missing context %s", t.ID, t.Context)
			}
		}
	}

	for _, g := range wf.Gateways {
		if g.ID == "" {
			return fmt.Errorf("gateway id cannot be empty")
		}
		if _, exists := gatewayIDs[g.ID]; exists {
			return fmt.Errorf("duplicate gateway id: %s", g.ID)
		}
		gatewayIDs[g.ID] = struct{}{}

		for _, wait := range append(g.Inputs, g.WaitFor...) {
			if wait == "" {
				return fmt.Errorf("gateway %s has empty input/wait_for entry", g.ID)
			}
			if _, ok := taskIDs[wait]; !ok {
				return fmt.Errorf("gateway %s references missing task %s", g.ID, wait)
			}
		}
	}

	return nil
}
