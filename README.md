# Petri Net Workflow Engine - MVP

A demonstration of Petri net-based workflow execution in Go, showcasing advantages over traditional DAG approaches for specific use cases.

## What is a Petri Net?

A Petri net is a mathematical modeling language for describing distributed systems:
- **Places** = States (can hold tokens)
- **Transitions** = Actions (can fire when enabled)
- **Tokens** = Data/Resources flowing through
- **Arcs** = Connections between places and transitions

## Why Petri Nets for Workflows?

### Advantages over DAG

1. **Natural Resource Constraints** - Model API rate limits, connection pools without semaphores
2. **Built-in Synchronization** - Barrier patterns, producer-consumer without explicit locks
3. **Formal Semantics** - Can prove properties (no deadlocks, liveness)
4. **Concurrent by Design** - Maximal parallelism automatically
5. **Backpressure Handling** - Bounded queues prevent overflow

### When to Use Petri Nets

✅ **Good for**:
- API rate limiting
- Resource pools (DB connections, workers)
- Producer-consumer patterns
- Complex synchronization
- Long-running workflows with checkpoints

❌ **Overkill for**:
- Simple linear workflows
- Basic iteration
- Tree-structured dependencies
- Most AI/LLM workflows

## Project Structure

```
petri-net-mvp/
├── core/
│   ├── place.go        # Places (hold tokens)
│   ├── transition.go   # Transitions (actions)
│   └── net.go          # Petri net orchestrator
├── examples/
│   ├── 01_rate_limiting.go      # API rate limiting demo
│   ├── 02_producer_consumer.go  # Bounded queue demo
│   └── 03_barrier_sync.go       # N-way synchronization
└── README.md
```

## Examples

### Example 1: API Rate Limiting

**Problem**: Process 10 API requests with max 3 concurrent calls.

**DAG Approach**:
```go
// Need explicit semaphore
sem := make(chan struct{}, 3)
for _, req := range requests {
    sem <- struct{}{}  // Acquire
    go func() {
        defer func() { <-sem }()  // Release
        callAPI(req)
    }()
}
```

**Petri Net Approach**:
```
Places:
  - api_tokens: 3 tokens (max concurrent)
  - pending_requests: queue of requests
  - completed: results

Transition:
  Input:  1 api_token + 1 request
  Action: Call API
  Output: 1 api_token + 1 result
```

✨ **Advantage**: Resource constraint is declarative, not imperative.

**Run**:
```bash
cd examples
go run 01_rate_limiting.go
```

**Output**:
```
🚀 Starting Petri Net: API Rate Limiter
  📡 Processing API request 0...
  📡 Processing API request 1...
  📡 Processing API request 2...
  ✅ Completed API request 0
  📡 Processing API request 3...
  ...
📊 Final State:
  [API Tokens]: 3 tokens
  [Completed]: 10 tokens
⏱️  Total time: ~2s
```

---

### Example 2: Producer-Consumer

**Problem**: Producer creates items faster than consumer processes them.

**DAG Approach**:
```go
// Need explicit channel with capacity
queue := make(chan Item, 5)  // Bounded queue

go func() {  // Producer
    for _, item := range items {
        queue <- item  // Blocks when full
    }
}()

go func() {  // Consumer
    for item := range queue {
        process(item)
    }
}()
```

**Petri Net Approach**:
```
Places:
  - producer_ready: 1 token
  - queue: max 5 tokens (bounded)
  - consumer_ready: 1 token

Transitions:
  Produce: ready → queue (+1 item) → ready
  Consume: ready + queue → processed → ready
```

✨ **Advantage**: Backpressure is automatic. Producer naturally slows when queue approaches capacity.

**Run**:
```bash
go run 02_producer_consumer.go
```

---

### Example 3: Barrier Synchronization

**Problem**: Wait for all 3 workers to complete before continuing.

**DAG Approach**:
```go
var wg sync.WaitGroup
wg.Add(3)

go func() { worker1(); wg.Done() }()
go func() { worker2(); wg.Done() }()
go func() { worker3(); wg.Done() }()

wg.Wait()  // Block until all done
processResults()
```

**Petri Net Approach**:
```
Places:
  - worker1_done, worker2_done, worker3_done

Transition (Barrier):
  Input: worker1_done + worker2_done + worker3_done
  Action: aggregate results
  Output: all_done
```

✨ **Advantage**: Synchronization is visual and declarative. Transition only fires when ALL inputs ready.

**Run**:
```bash
go run 03_barrier_sync.go
```

---

## Core API

### Creating a Petri Net

```go
net := core.NewPetriNet("My Workflow")

// Create places
input := core.NewPlace("input", "Input Queue", 10)  // Capacity 10
output := core.NewPlace("output", "Results", -1)    // Unlimited

// Create transition
process := core.NewTransition("process", "Process Item")
process.AddInputArc(input, 1)   // Consume 1 token
process.AddOutputArc(output, 1)  // Produce 1 token

process.Action = func(ctx context.Context, tokens []*core.Token) ([]*core.Token, error) {
    // Your logic here
    return []*core.Token{{ID: "result", Data: "processed"}}, nil
}

// Add to net
net.AddPlace(input)
net.AddPlace(output)
net.AddTransition(process)

// Run
ctx := context.Background()
net.Run(ctx)
```

### Guard Conditions

```go
transition.Guard = func(tokens []*core.Token) bool {
    // Only fire if condition met
    return tokens[0].Data.(int) > 10
}
```

### Continuous Execution

```go
// Run forever, firing transitions as they become enabled
ctx, cancel := context.WithCancel(context.Background())
net.RunContinuous(ctx, 100*time.Millisecond)
```

---

## Performance Characteristics

| Operation | Complexity |
|-----------|------------|
| Check if transition can fire | O(arcs) |
| Fire transition | O(arcs) |
| Find enabled transitions | O(transitions) |
| Overall execution | O(iterations × transitions × arcs) |

For workflows with many transitions, Petri nets can be slower than DAG execution. The trade-off is expressiveness vs raw speed.

---

## Comparison: Petri Net vs DAG

| Feature | DAG | Petri Net |
|---------|-----|-----------|
| **Simplicity** | ✅ Very simple | ⚠️ More complex |
| **Resource constraints** | ❌ Manual | ✅ Built-in |
| **Synchronization** | ⚠️ WaitGroup needed | ✅ Declarative |
| **Backpressure** | ❌ Channels/semaphores | ✅ Automatic |
| **Deadlock detection** | ❌ Hard to prove | ✅ Analyzable |
| **Visual clarity** | ✅ Excellent | ✅ Excellent |
| **Performance** | ✅ Fast | ⚠️ Moderate |
| **Use case fit (AI)** | ✅ Perfect | ⚠️ Overkill |

---

## Recommendation

**For Dify workflows**: Stick with DAG for 95% of cases.

**Add Petri net semantics** for:
- API rate limiting (ChatGPT, Claude quotas)
- Database connection pools
- Worker pool management
- Complex approval workflows

**Hybrid approach**: Use DAG syntax, Petri net engine for resource management internally.

---

## Build & Run

```bash
# Build examples
cd examples
go build -o rate_limit 01_rate_limiting.go
go build -o prod_cons 02_producer_consumer.go
go build -o barrier 03_barrier_sync.go

# Run
./rate_limit
./prod_cons
./barrier
```

---

## Future Enhancements

- [ ] YAML-based Petri net definition
- [ ] Deadlock detection algorithm
- [ ] Reachability analysis
- [ ] State space visualization
- [ ] Checkpointing and replay
- [ ] Timed transitions
- [ ] Colored tokens (typed data)

---

## References

- [Petri Nets - Wikipedia](https://en.wikipedia.org/wiki/Petri_net)
- [Workflow Nets](https://en.wikipedia.org/wiki/Workflow_net)
- van der Aalst, W.M.P. (1998). "The Application of Petri Nets to Workflow Management"

---

**Author**: Dify Go MVP Project  
**License**: MIT  
**Purpose**: Educational demonstration of Petri net advantages for workflow systems
