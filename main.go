package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          PETRI NET WORKFLOW ENGINE - DEMO                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This demo showcases Petri net advantages over DAG workflows:")
	fmt.Println()
	fmt.Println("1. API Rate Limiting    - Natural resource constraints")
	fmt.Println("2. Producer-Consumer    - Automatic backpressure")
	fmt.Println("3. Barrier Sync         - Declarative N-way wait")
	fmt.Println()
	fmt.Print("Select example (1-3) or 'q' to quit: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	choice := scanner.Text()

	var example string
	switch choice {
	case "1":
		example = "examples/01_rate_limiting.go"
	case "2":
		example = "examples/02_producer_consumer.go"
	case "3":
		example = "examples/03_barrier_sync.go"
	case "q", "Q":
		fmt.Println("Goodbye!")
		return
	default:
		fmt.Println("Invalid choice")
		return
	}

	fmt.Println()
	fmt.Println("Running example...")
	fmt.Println()

	cmd := exec.Command("go", "run", example)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running example: %v\n", err)
	}
}
