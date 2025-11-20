package main

import (
	"context"
	"fmt"
	"petri-net-mvp/core/petrinet"
	"strings"
	"time"
)

// Example 1: API Rate Limiting
// Demonstrates: Natural resource constraints without semaphores

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("EXAMPLE 1: API RATE LIMITING WITH PETRI NET")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Scenario: Process 10 API requests with max 3 concurrent calls")
	fmt.Println("")

	// Create Petri net
	net := petrinet.NewPetriNet("API Rate Limiter")

	// Places
	apiTokens := petrinet.NewPlace("api_tokens", "API Tokens", 3)
	pendingRequests := petrinet.NewPlace("pending", "Pending Requests", -1)
	completedRequests := petrinet.NewPlace("completed", "Completed", -1)

	// Initialize: 3 API tokens (max concurrent)
	for i := 0; i < 3; i++ {
		apiTokens.AddTokens(&petrinet.Token{ID: fmt.Sprintf("token-%d", i)})
	}

	// Initialize: 10 pending requests
	for i := 0; i < 10; i++ {
		pendingRequests.AddTokens(&petrinet.Token{
			ID:   fmt.Sprintf("req-%d", i),
			Data: map[string]interface{}{"request_id": i},
		})
	}

	net.AddPlace(apiTokens)
	net.AddPlace(pendingRequests)
	net.AddPlace(completedRequests)

	// Transition: Make API call
	apiCall := petrinet.NewTransition("api_call", "Make API Call")
	apiCall.AddInputArc(apiTokens, 1)       // Need 1 token to proceed
	apiCall.AddInputArc(pendingRequests, 1) // Need 1 request
	apiCall.AddOutputArc(completedRequests, 1)
	apiCall.AddOutputArc(apiTokens, 1) // Return token after completion

	apiCall.Action = func(ctx context.Context, tokens []*petrinet.Token) ([]*petrinet.Token, error) {
		reqToken := tokens[1] // Second token is the request
		reqData := reqToken.Data.(map[string]interface{})
		reqID := reqData["request_id"]

		fmt.Printf("  📡 Processing API request %d...\n", reqID)
		time.Sleep(500 * time.Millisecond) // Simulate API call
		fmt.Printf("  ✅ Completed API request %d\n", reqID)

		// Return token and completion marker
		return []*petrinet.Token{
			tokens[0], // API token (returned)
			&petrinet.Token{ID: fmt.Sprintf("result-%d", reqID), Data: "success"},
		}, nil
	}

	net.AddTransition(apiCall)

	// Run the net
	ctx := context.Background()
	startTime := time.Now()

	if err := net.Run(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("\n⏱️  Total time: %v\n", duration)
	fmt.Printf("📊 Expected time: ~2s (10 requests / 3 concurrent ≈ 4 batches × 0.5s)\n")
	fmt.Printf("✨ Petri net naturally enforces rate limit without explicit semaphore!\n")
}
