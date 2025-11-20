package petrinet

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PetriNet orchestrates places and transitions
type PetriNet struct {
	Name        string
	Places      map[string]*Place
	Transitions map[string]*Transition
	mu          sync.RWMutex
}

// NewPetriNet creates a new Petri net
func NewPetriNet(name string) *PetriNet {
	return &PetriNet{
		Name:        name,
		Places:      make(map[string]*Place),
		Transitions: make(map[string]*Transition),
	}
}

// AddPlace adds a place to the net
func (pn *PetriNet) AddPlace(place *Place) {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.Places[place.ID] = place
}

// AddTransition adds a transition to the net
func (pn *PetriNet) AddTransition(transition *Transition) {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.Transitions[transition.ID] = transition
}

// Run executes the Petri net until no transitions can fire
func (pn *PetriNet) Run(ctx context.Context) error {
	fmt.Printf("🚀 Starting Petri Net: %s\n", pn.Name)

	iterations := 0
	maxIterations := 1000 // Prevent infinite loops

	for iterations < maxIterations {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Find all transitions that can fire
		var fireable []*Transition
		pn.mu.RLock()
		for _, t := range pn.Transitions {
			if t.CanFire() {
				fireable = append(fireable, t)
			}
		}
		pn.mu.RUnlock()

		if len(fireable) == 0 {
			fmt.Printf("✅ No more transitions can fire. Completed in %d iterations.\n", iterations)
			break
		}

		// Fire all enabled transitions (maximal concurrency)
		var wg sync.WaitGroup
		errorsChan := make(chan error, len(fireable))

		for _, t := range fireable {
			wg.Add(1)
			go func(transition *Transition) {
				defer wg.Done()
				if err := transition.Fire(ctx); err != nil {
					errorsChan <- fmt.Errorf("%s: %w", transition.Name, err)
				} else {
					fmt.Printf("  🔥 Fired: %s\n", transition.Name)
				}
			}(t)
		}

		wg.Wait()
		close(errorsChan)

		// Check for errors
		for err := range errorsChan {
			if err != nil {
				return err
			}
		}

		iterations++
		time.Sleep(10 * time.Millisecond) // Small delay for readability
	}

	if iterations >= maxIterations {
		return fmt.Errorf("reached max iterations (%d), possible infinite loop", maxIterations)
	}

	pn.PrintState()
	return nil
}

// PrintState shows current state of all places
func (pn *PetriNet) PrintState() {
	fmt.Println("\n📊 Final State:")
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	for _, place := range pn.Places {
		fmt.Printf("  [%s]: %d tokens\n", place.Name, place.TokenCount())
	}
}

// RunContinuous runs the net continuously, firing transitions as they become enabled
func (pn *PetriNet) RunContinuous(ctx context.Context, pollInterval time.Duration) error {
	fmt.Printf("🔄 Starting Continuous Petri Net: %s\n", pn.Name)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("⏹️  Stopped")
			return ctx.Err()
		default:
		}

		// Find and fire one transition
		fired := false
		pn.mu.RLock()
		for _, t := range pn.Transitions {
			if t.CanFire() {
				pn.mu.RUnlock()
				if err := t.Fire(ctx); err == nil {
					fmt.Printf("  🔥 Fired: %s\n", t.Name)
					fired = true
				}
				pn.mu.RLock()
				break
			}
		}
		pn.mu.RUnlock()

		if !fired {
			time.Sleep(pollInterval)
		}
	}
}
