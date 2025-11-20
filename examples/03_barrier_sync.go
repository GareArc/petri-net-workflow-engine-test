package main

import (
	"context"
	"fmt"
	"petri-net-mvp/core/petrinet"
	"strings"
	"time"
)

// Example 3: Barrier Synchronization
// Demonstrates: N-way synchronization (wait for all workers)

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 3: BARRIER SYNCHRONIZATION")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Scenario: 3 workers must all complete before proceeding")
	fmt.Println("")

	net := petrinet.NewPetriNet("Barrier Synchronization")

	// Places
	worker1Ready := petrinet.NewPlace("w1_ready", "Worker 1 Ready", 1)
	worker2Ready := petrinet.NewPlace("w2_ready", "Worker 2 Ready", 1)
	worker3Ready := petrinet.NewPlace("w3_ready", "Worker 3 Ready", 1)

	worker1Done := petrinet.NewPlace("w1_done", "Worker 1 Done", 1)
	worker2Done := petrinet.NewPlace("w2_done", "Worker 2 Done", 1)
	worker3Done := petrinet.NewPlace("w3_done", "Worker 3 Done", 1)

	allDone := petrinet.NewPlace("all_done", "All Workers Done", 1)

	// Initialize: All workers ready
	worker1Ready.AddTokens(&petrinet.Token{ID: "w1"})
	worker2Ready.AddTokens(&petrinet.Token{ID: "w2"})
	worker3Ready.AddTokens(&petrinet.Token{ID: "w3"})

	net.AddPlace(worker1Ready)
	net.AddPlace(worker2Ready)
	net.AddPlace(worker3Ready)
	net.AddPlace(worker1Done)
	net.AddPlace(worker2Done)
	net.AddPlace(worker3Done)
	net.AddPlace(allDone)

	// Worker 1 task
	task1 := petrinet.NewTransition("task1", "Worker 1 Task")
	task1.AddInputArc(worker1Ready, 1)
	task1.AddOutputArc(worker1Done, 1)
	task1.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		fmt.Println("  ⚙️  Worker 1: Starting work...")
		time.Sleep(1 * time.Second)
		fmt.Println("  ✅ Worker 1: Completed!")
		return []*petrinet.Token{{ID: "w1-result", Data: "result1"}}, nil
	}
	net.AddTransition(task1)

	// Worker 2 task (slower)
	task2 := petrinet.NewTransition("task2", "Worker 2 Task")
	task2.AddInputArc(worker2Ready, 1)
	task2.AddOutputArc(worker2Done, 1)
	task2.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		fmt.Println("  ⚙️  Worker 2: Starting work...")
		time.Sleep(2 * time.Second)
		fmt.Println("  ✅ Worker 2: Completed!")
		return []*petrinet.Token{{ID: "w2-result", Data: "result2"}}, nil
	}
	net.AddTransition(task2)

	// Worker 3 task
	task3 := petrinet.NewTransition("task3", "Worker 3 Task")
	task3.AddInputArc(worker3Ready, 1)
	task3.AddOutputArc(worker3Done, 1)
	task3.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		fmt.Println("  ⚙️  Worker 3: Starting work...")
		time.Sleep(1500 * time.Millisecond)
		fmt.Println("  ✅ Worker 3: Completed!")
		return []*petrinet.Token{{ID: "w3-result", Data: "result3"}}, nil
	}
	net.AddTransition(task3)

	// Barrier transition: Only fires when ALL workers done
	barrier := petrinet.NewTransition("barrier", "Barrier (Wait for All)")
	barrier.AddInputArc(worker1Done, 1)
	barrier.AddInputArc(worker2Done, 1)
	barrier.AddInputArc(worker3Done, 1)
	barrier.AddOutputArc(allDone, 1)
	barrier.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		fmt.Println("\n  🎯 BARRIER REACHED: All workers completed!")
		fmt.Println("  📊 Aggregating results...")

		results := make([]interface{}, 0, len(tokens))
		for _, token := range tokens {
			results = append(results, token.Data)
		}

		fmt.Printf("  ✨ Combined results: %v\n", results)
		return []*petrinet.Token{{ID: "final", Data: results}}, nil
	}
	net.AddTransition(barrier)

	// Run
	ctx := context.Background()
	startTime := time.Now()

	if err := net.Run(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("\n⏱️  Total time: %v\n", duration)
	fmt.Printf("📊 Expected: ~2s (slowest worker)\n")
	fmt.Printf("\n✨ Petri net advantages:\n")
	fmt.Printf("  - Natural barrier (transition waits for all inputs)\n")
	fmt.Printf("  - No manual WaitGroup needed\n")
	fmt.Printf("  - Declarative synchronization\n")
}
