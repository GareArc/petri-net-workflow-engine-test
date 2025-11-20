package petrinet

import (
	"context"
	"fmt"
	"sync"
)

// Arc represents a connection between a place and a transition
type Arc struct {
	Place  *Place
	Weight int // Number of tokens to consume/produce
}

// Transition represents an action that can fire
type Transition struct {
	ID         string
	Name       string
	InputArcs  []*Arc
	OutputArcs []*Arc
	Guard      func([]*Token) bool // Optional guard condition
	Action     func(context.Context, []*Token) ([]*Token, error)
	mu         sync.Mutex
}

// NewTransition creates a new transition
func NewTransition(id, name string) *Transition {
	return &Transition{
		ID:         id,
		Name:       name,
		InputArcs:  make([]*Arc, 0),
		OutputArcs: make([]*Arc, 0),
	}
}

// AddInputArc adds an input arc (place → transition)
func (t *Transition) AddInputArc(place *Place, weight int) {
	t.InputArcs = append(t.InputArcs, &Arc{Place: place, Weight: weight})
}

// AddOutputArc adds an output arc (transition → place)
func (t *Transition) AddOutputArc(place *Place, weight int) {
	t.OutputArcs = append(t.OutputArcs, &Arc{Place: place, Weight: weight})
}

// CanFire checks if transition can fire (enough tokens in all input places)
func (t *Transition) CanFire() bool {
	for _, arc := range t.InputArcs {
		if arc.Place.TokenCount() < arc.Weight {
			return false
		}
	}
	return true
}

// Fire executes the transition
func (t *Transition) Fire(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if can fire
	if !t.CanFire() {
		return fmt.Errorf("transition %s cannot fire (not enough tokens)", t.Name)
	}

	// Collect input tokens
	var inputTokens []*Token
	for _, arc := range t.InputArcs {
		tokens, err := arc.Place.RemoveTokens(arc.Weight)
		if err != nil {
			// Rollback: put back already removed tokens
			for i := len(t.InputArcs) - 1; i >= 0; i-- {
				prevArc := t.InputArcs[i]
				if prevArc == arc {
					break
				}
				prevArc.Place.AddTokens(inputTokens[:len(inputTokens)-prevArc.Weight]...)
				inputTokens = inputTokens[:len(inputTokens)-prevArc.Weight]
			}
			return err
		}
		inputTokens = append(inputTokens, tokens...)
	}

	// Check guard condition
	if t.Guard != nil && !t.Guard(inputTokens) {
		// Guard failed, put tokens back
		offset := 0
		for _, arc := range t.InputArcs {
			arc.Place.AddTokens(inputTokens[offset : offset+arc.Weight]...)
			offset += arc.Weight
		}
		return fmt.Errorf("guard condition failed for %s", t.Name)
	}

	// Execute action
	var outputTokens []*Token
	var err error
	if t.Action != nil {
		outputTokens, err = t.Action(ctx, inputTokens)
		if err != nil {
			// Action failed, put input tokens back
			offset := 0
			for _, arc := range t.InputArcs {
				arc.Place.AddTokens(inputTokens[offset : offset+arc.Weight]...)
				offset += arc.Weight
			}
			return fmt.Errorf("action failed for %s: %w", t.Name, err)
		}
	} else {
		// No action, pass through input tokens
		outputTokens = inputTokens
	}

	// Distribute output tokens
	tokenIdx := 0
	for _, arc := range t.OutputArcs {
		tokensToAdd := outputTokens[tokenIdx:min(tokenIdx+arc.Weight, len(outputTokens))]

		// If not enough output tokens, create generic ones
		for len(tokensToAdd) < arc.Weight {
			tokensToAdd = append(tokensToAdd, &Token{ID: fmt.Sprintf("gen-%d", len(tokensToAdd))})
		}

		if err := arc.Place.AddTokens(tokensToAdd...); err != nil {
			return fmt.Errorf("failed to add output tokens to %s: %w", arc.Place.Name, err)
		}
		tokenIdx += arc.Weight
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
