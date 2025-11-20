package main

import (
	"context"
	"fmt"
	"petri-net-mvp/core/petrinet"
	"strings"
	"time"
)

// Example 2: Producer-Consumer with Bounded Queue
// Demonstrates: Backpressure handling and deadlock prevention

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 2: PRODUCER-CONSUMER WITH BOUNDED QUEUE")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Scenario: Producer creates items, consumer processes them")
	fmt.Println("Queue capacity: 5 items (demonstrates backpressure)")
	fmt.Println("")

	net := petrinet.NewPetriNet("Producer-Consumer")

	// Places
	producerReady := petrinet.NewPlace("producer_ready", "Producer Ready", 1)
	queue := petrinet.NewPlace("queue", "Item Queue", 5) // Max 5 items
	consumerReady := petrinet.NewPlace("consumer_ready", "Consumer Ready", 1)
	processed := petrinet.NewPlace("processed", "Processed Items", -1)

	// Initialize
	producerReady.AddTokens(&petrinet.Token{ID: "producer"})
	consumerReady.AddTokens(&petrinet.Token{ID: "consumer"})

	net.AddPlace(producerReady)
	net.AddPlace(queue)
	net.AddPlace(consumerReady)
	net.AddPlace(processed)

	// Producer transition
	itemsProduced := 0
	maxItems := 10

	produce := petrinet.NewTransition("produce", "Produce Item")
	produce.AddInputArc(producerReady, 1)
	produce.AddOutputArc(queue, 1)         // Put item in queue
	produce.AddOutputArc(producerReady, 1) // Producer stays ready

	produce.Guard = func(tokens []*petrinet.Token) bool {
		return itemsProduced < maxItems // Stop after 10 items
	}

	produce.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		itemsProduced++
		fmt.Printf("  🏭 Producer: Created item %d (queue: %d/%d)\n", itemsProduced, queue.TokenCount(), 5)
		time.Sleep(100 * time.Millisecond)

		return []*petrinet.Token{
			&petrinet.Token{ID: fmt.Sprintf("item-%d", itemsProduced), Data: itemsProduced},
			tokens[0], // Producer token
		}, nil
	}

	net.AddTransition(produce)

	// Consumer transition
	consume := petrinet.NewTransition("consume", "Consume Item")
	consume.AddInputArc(consumerReady, 1)
	consume.AddInputArc(queue, 1) // Take item from queue
	consume.AddOutputArc(processed, 1)
	consume.AddOutputArc(consumerReady, 1) // Consumer stays ready

	consume.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		item := tokens[1]
		fmt.Printf("  📦 Consumer: Processing %s (queue: %d/%d)\n", item.ID, queue.TokenCount(), 5)
		time.Sleep(150 * time.Millisecond) // Slower than producer

		return []*petrinet.Token{
			&petrinet.Token{ID: fmt.Sprintf("result-%s", item.ID), Data: "processed"},
			tokens[0], // Consumer token
		}, nil
	}

	net.AddTransition(consume)

	// Run continuously with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(4 * time.Second)
		cancel()
	}()

	fmt.Println("🔄 Running for 4 seconds...")
	net.RunContinuous(ctx, 50*time.Millisecond)

	fmt.Println("\n✨ Petri net automatically handles:")
	fmt.Println("  - Backpressure (producer slows when queue full)")
	fmt.Println("  - Synchronization (consumer waits for items)")
	fmt.Println("  - No deadlock (bounded queue prevents overflow)")
}
