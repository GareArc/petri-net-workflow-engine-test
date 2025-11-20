package workflow

import (
	"context"
	"fmt"
	"petri-net-mvp/core/petrinet"
)

// Compiler converts high-level Workflow to low-level Petri net
type Compiler struct{}

// NewCompiler creates a new workflow compiler
func NewCompiler() *Compiler {
	return &Compiler{}
}

// Compile transforms a Workflow into a Petri net
func (c *Compiler) Compile(wf *Workflow) (*petrinet.PetriNet, error) {
	net := petrinet.NewPetriNet(wf.Name)

	// Step 1: Create places for resources
	for _, resource := range wf.Resources {
		place := petrinet.NewPlace(
			resource.ID,
			resource.ID,
			resource.Capacity,
		)

		// Initialize with tokens (capacity = initial tokens)
		if resource.Capacity > 0 {
			for i := 0; i < resource.Capacity; i++ {
				place.AddTokens(&petrinet.Token{
					ID:   fmt.Sprintf("%s-token-%d", resource.ID, i),
					Data: resource.ID,
				})
			}
		}

		net.AddPlace(place)
	}

	// Step 2: Create places for contexts
	for _, ctx := range wf.Contexts {
		place := petrinet.NewPlace(
			ctx.ID,
			ctx.ID,
			ctx.Capacity,
		)
		// Seed with a single context token if capacity allows.
		if ctx.Capacity != 0 {
			place.AddTokens(&petrinet.Token{
				ID:   fmt.Sprintf("%s-token-0", ctx.ID),
				Data: map[string]interface{}{},
			})
		}
		net.AddPlace(place)
	}

	// Step 3: Create places for channels
	for _, channel := range wf.Channels {
		place := petrinet.NewPlace(
			channel.ID,
			channel.ID,
			channel.Capacity,
		)
		net.AddPlace(place)
	}

	// Step 4: Create transitions for tasks
	for _, task := range wf.Tasks {
		transition := c.compileTask(task)
		net.AddTransition(transition)

		// Connect context place if specified (consumed and re-emitted).
		if task.Context != "" {
			if ctxPlace, ok := net.Places[task.Context]; ok {
				transition.AddInputArc(ctxPlace, 1)
				transition.AddOutputArc(ctxPlace, 1)
			} else {
				return nil, fmt.Errorf("task %s references missing context place %s", task.ID, task.Context)
			}
		}

		// Connect input channels
		if task.Input != "" {
			transition.AddInputArc(net.Places[task.Input], 1)
		}
		for _, inputID := range task.Inputs {
			transition.AddInputArc(net.Places[inputID], 1)
		}

		// Connect resource requirements
		for resourceID, amount := range task.Requires {
			if place, exists := net.Places[resourceID]; exists {
				transition.AddInputArc(place, amount)
				transition.AddOutputArc(place, amount) // Return resource after use
			}
		}

		// Connect output channels
		if task.Output != "" {
			transition.AddOutputArc(net.Places[task.Output], 1)
		}
		for _, outputID := range task.Outputs {
			transition.AddOutputArc(net.Places[outputID], 1)
		}

		// Emit completion signal for barrier gateways.
		donePlaceID := task.ID + "_done"
		if _, exists := net.Places[donePlaceID]; !exists {
			net.AddPlace(petrinet.NewPlace(donePlaceID, task.ID+" Done", -1))
		}
		transition.AddOutputArc(net.Places[donePlaceID], 1)
	}

	// Step 5: Handle gateways (barriers, splits, merges)
	for _, gateway := range wf.Gateways {
		if err := c.compileGateway(gateway, net); err != nil {
			return nil, err
		}
	}

	return net, nil
}

// compileTask converts a Task to a Petri net Transition
func (c *Compiler) compileTask(task Task) *petrinet.Transition {
	transition := petrinet.NewTransition(task.ID, task.ID)

	// Wrap task action to handle Petri net token inputs/outputs
	transition.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		// Extract input data from tokens
		var inputData interface{}
		if len(tokens) > 0 {
			// Skip resource tokens, find data token
			for _, token := range tokens {
				if token.Data != nil {
					inputData = token.Data
					break
				}
			}
		}

		// Execute task action
		var outputData interface{}
		var err error
		if task.Action != nil {
			outputData, err = task.Action(ctx, inputData)
			if err != nil {
				return nil, err
			}
		} else {
			// No action, pass through
			outputData = inputData
		}

		// Create output tokens
		outputTokens := make([]*petrinet.Token, 0)

		// Add resource tokens back (already in input tokens)
		for _, token := range tokens {
			// Check if this is a resource token
			isResource := false
			for resourceID := range task.Requires {
				if token.Data == resourceID {
					isResource = true
					break
				}
			}
			if isResource {
				outputTokens = append(outputTokens, token)
			}
		}

		// Add data output token
		if outputData != nil {
			outputTokens = append(outputTokens, &petrinet.Token{
				ID:   fmt.Sprintf("%s-output", task.ID),
				Data: outputData,
			})
		}

		return outputTokens, nil
	}

	return transition
}

// compileGateway converts a Gateway to Petri net structures
func (c *Compiler) compileGateway(gateway Gateway, net *petrinet.PetriNet) error {
	switch gateway.Type {
	case "barrier":
		// Barrier: Wait for all inputs before proceeding
		waitFor := gateway.Inputs
		if len(waitFor) == 0 {
			waitFor = gateway.WaitFor
		}
		if len(waitFor) == 0 {
			return nil
		}

		// Create a place for barrier state
		barrierPlace := petrinet.NewPlace(
			gateway.ID+"_complete",
			gateway.ID+" Complete",
			1,
		)
		net.AddPlace(barrierPlace)

		// Create transition that fires when all inputs ready
		barrierTransition := petrinet.NewTransition(gateway.ID, gateway.ID)

		// Add input arcs from all waited tasks (via *_done places)
		for _, waitID := range waitFor {
			placeID := waitID + "_done"
			signalPlace, ok := net.Places[placeID]
			if !ok {
				return fmt.Errorf("barrier %s waits on missing completion place %s", gateway.ID, placeID)
			}
			barrierTransition.AddInputArc(signalPlace, 1)
		}

		barrierTransition.AddOutputArc(barrierPlace, 1)
		net.AddTransition(barrierTransition)
	}

	return nil
}
