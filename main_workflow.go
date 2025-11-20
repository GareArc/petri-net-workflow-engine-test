package main

import (
	"context"
	"fmt"
	"petri-net-mvp/core/petrinet"
	"petri-net-mvp/core/workflow"
	"petri-net-mvp/dsl"
	"time"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║        PETRI NET WORKFLOW ENGINE - DSL DEMO               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Parse workflow from YAML
	parser := dsl.NewParser()
	wf, err := parser.ParseFile("workflows/api_rate_limit.yml")
	if err != nil {
		fmt.Printf("Error parsing workflow: %v\n", err)
		return
	}

	fmt.Printf("📋 Loaded workflow: %s\n", wf.Name)
	fmt.Printf("   Resources: %d\n", len(wf.Resources))
	fmt.Printf("   Channels:  %d\n", len(wf.Channels))
	fmt.Printf("   Tasks:     %d\n", len(wf.Tasks))
	fmt.Println()

	// Compile workflow to Petri net
	compiler := workflow.NewCompiler()
	net, err := compiler.Compile(wf)
	if err != nil {
		fmt.Printf("Error compiling workflow: %v\n", err)
		return
	}

	fmt.Println("✅ Workflow compiled to Petri net")
	fmt.Printf("   Places:      %d\n", len(net.Places))
	fmt.Printf("   Transitions: %d\n", len(net.Transitions))
	fmt.Println()

	// Simulate with mock data
	fmt.Println("🚀 Simulating workflow with mock data...")
	fmt.Println()

	// Add mock documents to input channel
	docsPlace := net.Places["documents"]
	for i := 0; i < 10; i++ {
		docsPlace.AddTokens(&petrinet.Token{
			ID:   fmt.Sprintf("doc-%d", i),
			Data: fmt.Sprintf("Document content %d", i),
		})
	}

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := net.Run(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("\n✨ Workflow completed successfully!")
	fmt.Println("\n🔑 Key Advantages:")
	fmt.Println("   ✅ YAML-based workflow definition (no code!)")
	fmt.Println("   ✅ Natural resource constraints (API tokens)")
	fmt.Println("   ✅ Automatic rate limiting via Petri net")
	fmt.Println("   ✅ High-level abstractions (tasks, not transitions)")
}
