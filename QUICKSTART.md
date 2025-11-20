# Petri Net MVP - Quick Start

## Project Structure

```
/Users/gareth/Documents/Code/dify/petri-net-mvp/
├── core/                 # Petri net engine
│   ├── net.go           # Net orchestrator
│   ├── place.go         # Places (hold tokens)
│   └── transition.go    # Transitions (actions)
├── examples/
│   ├── 01_rate_limiting.go      # API rate limiting
│   ├── 02_producer_consumer.go  # Bounded queue
│   └── 03_barrier_sync.go       # N-way synchronization
├── main.go              # Interactive demo runner
└── README.md            # Full documentation
```

## Running Examples

```bash
cd /Users/gareth/Documents/Code/dify/petri-net-mvp

# Example 1: API Rate Limiting
go run examples/01_rate_limiting.go

# Example 2: Producer-Consumer
go run examples/02_producer_consumer.go

# Example 3: Barrier Synchronization
go run examples/03_barrier_sync.go

# Interactive demo
go run main.go
```

## Key Advantages Demonstrated

1. **API Rate Limiting** - 3 API tokens = max 3 concurrent calls, naturally enforced
2. **Producer-Consumer** - Bounded queue (5 items) with automatic backpressure
3. **Barrier Sync** - Wait for all 3 workers without manual WaitGroup

## vs DAG Approach

Petri nets excel when you need:
- Resource constraints (API quotas, connection pools)
- Backpressure (bounded queues)
- Complex synchronization (barriers, rendez-vous)

DAG is better for:
- Simple linear/tree workflows
- Most AI/LLM use cases
- Iteration patterns

## Next Steps

See full `README.md` for:
- Detailed API documentation
- Performance characteristics
- Comparison table: Petri Net vs DAG
- Future enhancements
