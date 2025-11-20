package petrinet

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	// ErrNotReady indicates a transition cannot currently fire (insufficient tokens, guard fail, or no output capacity).
	ErrNotReady = errors.New("transition not ready")
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

	// Collect unique places involved (inputs + outputs) and lock deterministically to avoid races/deadlocks.
	placeSet := make(map[*Place]struct{})
	for _, arc := range t.InputArcs {
		placeSet[arc.Place] = struct{}{}
	}
	for _, arc := range t.OutputArcs {
		placeSet[arc.Place] = struct{}{}
	}

	orderedPlaces := make([]*Place, 0, len(placeSet))
	for p := range placeSet {
		orderedPlaces = append(orderedPlaces, p)
	}
	sort.Slice(orderedPlaces, func(i, j int) bool { return orderedPlaces[i].ID < orderedPlaces[j].ID })

	for _, p := range orderedPlaces {
		p.mu.Lock()
	}
	defer func() {
		for i := len(orderedPlaces) - 1; i >= 0; i-- {
			orderedPlaces[i].mu.Unlock()
		}
	}()

	// Build counts per place.
	inputCounts := make(map[*Place]int)
	for _, arc := range t.InputArcs {
		inputCounts[arc.Place] += arc.Weight
	}
	outputCounts := make(map[*Place]int)
	for _, arc := range t.OutputArcs {
		outputCounts[arc.Place] += arc.Weight
	}

	// Check input availability and output capacity (accounting for tokens that will be consumed then returned to same place).
	for place, need := range inputCounts {
		if len(place.Tokens) < need {
			return ErrNotReady
		}
	}
	for place, outNeed := range outputCounts {
		current := len(place.Tokens)
		inNeed := inputCounts[place]
		effective := current - inNeed + outNeed
		if place.Capacity >= 0 && effective > place.Capacity {
			return ErrNotReady
		}
	}

	// Consume input tokens.
	var inputTokens []*Token
	consumedPerPlace := make(map[*Place][]*Token)
	for _, arc := range t.InputArcs {
		tokens := arc.Place.Tokens[:arc.Weight]
		arc.Place.Tokens = arc.Place.Tokens[arc.Weight:]
		consumedPerPlace[arc.Place] = append(consumedPerPlace[arc.Place], tokens...)
		inputTokens = append(inputTokens, tokens...)
	}

	// Guard check: guard failure is treated as not-ready; return tokens and exit quietly.
	if t.Guard != nil && !t.Guard(inputTokens) {
		for place, tokens := range consumedPerPlace {
			place.Tokens = append(tokens, place.Tokens...)
		}
		return ErrNotReady
	}

	// Execute action.
	var outputTokens []*Token
	if t.Action != nil {
		actionOutput, err := t.Action(ctx, inputTokens)
		if err != nil {
			// Roll back consumed tokens on action failure.
			for place, tokens := range consumedPerPlace {
				place.Tokens = append(tokens, place.Tokens...)
			}
			return fmt.Errorf("action failed for %s: %w", t.Name, err)
		}
		outputTokens = actionOutput
	}

	// Return resource tokens first (places that were both consumed and produced).
	resourcePlaces := make(map[*Place]struct{})
	for place := range outputCounts {
		if _, consumed := inputCounts[place]; consumed {
			resourcePlaces[place] = struct{}{}
		}
	}
	for place := range resourcePlaces {
		outputTokens = append(outputTokens, consumedPerPlace[place]...)
	}

	// If no action was defined, pass through non-resource tokens.
	if t.Action == nil {
		for place, tokens := range consumedPerPlace {
			if _, isResource := resourcePlaces[place]; isResource {
				continue
			}
			outputTokens = append(outputTokens, tokens...)
		}
	}

	// Ensure output slice has enough tokens for all output arcs.
	totalNeeded := 0
	for _, need := range outputCounts {
		totalNeeded += need
	}
	if len(outputTokens) < totalNeeded {
		for len(outputTokens) < totalNeeded {
			outputTokens = append(outputTokens, &Token{ID: fmt.Sprintf("gen-%d", len(outputTokens))})
		}
	}

	// Distribute output tokens.
	offset := 0
	for _, arc := range t.OutputArcs {
		tokensToAdd := outputTokens[offset : offset+arc.Weight]
		arc.Place.Tokens = append(arc.Place.Tokens, tokensToAdd...)
		offset += arc.Weight
	}

	return nil
}
