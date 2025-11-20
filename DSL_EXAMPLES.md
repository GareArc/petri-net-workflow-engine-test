# Petri Net Workflow DSL Examples

This document explains the YAML-based DSL that powers the Petri net workflow compiler in `petri-net-mvp`. It walks through the vocabulary of the language and then dissects two end-to-end examples that already live in `workflows/`. Each section calls out how the DSL elements map to Petri net places, transitions, and tokens so you can adapt the patterns to your own workflows.

## DSL Anatomy at a Glance

Every DSL file follows the same structure:

```yaml
workflow:
  name: Human friendly label

  resources:
    - id: api_tokens
      type: semaphore      # Declares a place with capacity
      capacity: 3          # Number of resource tokens to preload

  channels:
    - id: documents
      capacity: 100        # Queue size (-1 = unlimited)
      type: fifo

  tasks:
    - id: process_doc
      type: llm            # Used by the runtime to pick an action
      input: documents     # Consumes from a channel place
      output: results      # Produces to a channel place
      requires:
        api_tokens: 1      # Also consumes/returns resource tokens

  gateways:
    - id: sync_barrier
      type: barrier
      wait_for: [task_a, task_b]
```

| Section    | Purpose                                                                 | Petri net mapping                                                      |
|------------|-------------------------------------------------------------------------|------------------------------------------------------------------------|
| `resources`| Declare shared capacity constraints (API quotas, worker pools).         | Places preloaded with `capacity` tokens.                              |
| `channels` | Model data flow queues between tasks with optional capacity limits.     | Places without initial tokens; capacity enforces bounded queues.      |
| `tasks`    | Describe units of work plus IO edges and resource requirements.         | Transitions with arcs to/from channel places and resource places.     |
| `gateways` | Express control-flow constructs such as barriers, splits, or merges.    | Adds helper places/transitions that coordinate other transitions.     |

The compiler (`core/workflow/compiler.go`) turns these declarations into places and transitions automatically, so the DSL author thinks in terms of tasks and resources rather than Petri net primitives.

---

## Example 1 – API Rate-Limited Document Processing

**File**: `workflows/api_rate_limit.yml`

```yaml
workflow:
  name: API Rate-Limited Document Processing

  resources:
    - id: api_tokens
      type: semaphore
      capacity: 3

  channels:
    - id: documents
      capacity: 100
      type: fifo
    - id: results
      capacity: -1

  tasks:
    - id: load_docs
      type: producer
      output: documents
      source: "./data/*.pdf"

    - id: process_doc
      type: llm
      input: documents
      output: results
      model: gpt-4
      prompt: "Summarize this document: {{input}}"
      requires:
        api_tokens: 1
      parallel: true

    - id: save_results
      type: consumer
      input: results
      destination: "./output/summaries.json"
```

### How the DSL Compiles

- **Resource declaration**  
  `api_tokens` becomes a place initialized with three tokens (`api_tokens-token-0`, etc.). Every firing of `process_doc` must consume one of these tokens and return it when the task completes. The Petri net enforces the quota without writing any semaphore logic.

- **Channel places**  
  `documents` and `results` each become places. `load_docs` emits tokens into `documents`, `process_doc` consumes one token per firing, and `save_results` drains the `results` place at its own pace. Because `results` has `capacity: -1`, it is effectively unbounded.

- **Task transitions**  
  Each task becomes a transition. `process_doc` gets:
  - Input arcs from `documents` and `api_tokens`.
  - Output arcs to `results` and back to `api_tokens`.
  - A wrapped action that would pass the channel payload to the configured LLM.

### Execution Story

1. `load_docs` fires repeatedly to tokenize every document path into the `documents` place.
2. Up to three `process_doc` transitions can be enabled simultaneously because of the `api_tokens` capacity. This produces natural backpressure when the API quota is exhausted.
3. `save_results` consumes whichever completed summaries are ready and writes them using the configured destination.

### Why It Matters

- **Declarative rate limiting**: No mutexes or buffered channels; the Petri net’s token accounting does it.
- **Separation of concerns**: YAML authors only describe resources and tasks, while Go code in `core/petrinet` handles orchestration.
- **Parallel safety**: Setting `parallel: true` signals to the runtime that multiple instances of the task may run concurrently, but the resource requirement still caps concurrency.

Run the example with:

```bash
cd petri-net-mvp
go run main_workflow.go
```

`main_workflow.go` loads `workflows/api_rate_limit.yml`, compiles it through the DSL parser, and executes the resulting Petri net.

---

## Example 2 – Parallel Pipeline With Barrier Synchronization

**File**: `workflows/pipeline_barrier.yml`

```yaml
workflow:
  name: Parallel Pipeline with Barrier

  channels:
    - id: raw_data
      capacity: 1
    - id: batch_a
      capacity: 1
    - id: batch_b
      capacity: 1
    - id: batch_c
      capacity: 1
    - id: result_a
      capacity: 1
    - id: result_b
      capacity: 1
    - id: result_c
      capacity: 1
    - id: final_result
      capacity: 1

  tasks:
    - id: fetch
      type: http
      output: raw_data
      config:
        url: "https://api.example.com/data"

    - id: split
      type: splitter
      input: raw_data
      outputs: [batch_a, batch_b, batch_c]

    - id: process_a
      type: transform
      input: batch_a
      output: result_a
      config: { script: "process_batch.py" }
    - id: process_b
      type: transform
      input: batch_b
      output: result_b
      config: { script: "process_batch.py" }
    - id: process_c
      type: transform
      input: batch_c
      output: result_c
      config: { script: "process_batch.py" }

    - id: merge
      type: aggregator
      inputs: [result_a, result_b, result_c]
      output: final_result

  gateways:
    - id: sync_barrier
      type: barrier
      wait_for: [process_a, process_b, process_c]
```

### How the DSL Compiles

- The `split` task becomes a transition with one input arc (`raw_data`) and multiple output arcs. Each firing duplicates the token into the three batch places, illustrating fan-out without manual bookkeeping.
- `process_a/b/c` are independent transitions consuming their respective batches and producing results, so they can run concurrently.
- The `sync_barrier` gateway expresses that all three processing tasks must complete before downstream work proceeds. The current compiler materializes a `<gateway_id>_complete` place; you can extend the DSL/compiler to add arcs from that place into tasks that should respect the barrier.
- The final `merge` task consumes one token from each of the `result_*` channel places and emits a single `final_result`. Because it waits on all three inputs, it effectively acts as a join/fan-in even without the barrier signal.

### Execution Story

1. `fetch` pulls remote data and drops it into `raw_data`.
2. `split` broadcasts the payload to three batch places—one firing of the transition produces three downstream tokens.
3. `process_a/b/c` run in parallel, limited only by their channel capacities.
4. `sync_barrier` fires once all three processing tasks have completed, signalling safe continuation.
5. `merge` aggregates the partial outputs into a final artifact.

### Why It Matters

- **Barrier semantics without code**: Even though the current compiler does not yet wire the barrier output automatically, the DSL captures the intent so the engine (or future extensions) can enforce it.
- **Explicit fan-out/fan-in**: Multiple `outputs` and `inputs` map to the many-to-many Petri net edges automatically.
- **Composable stages**: Because every step reads/writes named channels, you can insert additional tasks (validation, enrichment, etc.) by editing YAML alone.

---

## How to Craft Your Own DSL Workflows

1. **Identify resources and capacities** – Anything that must be rate limited (API keys, thread pools, GPU slots) should become a `resource`. Pick a `capacity` that matches the real-world quota and let the Petri net enforce it.
2. **Define channel boundaries** – Each channel carves out a queue between producer and consumer tasks. Use `capacity: -1` for unbounded throughput or a positive integer to impose backpressure.
3. **Describe tasks declaratively** – Give every task an `id`, a `type` (for runtime binding), and the relevant `input`/`output` fields. Add `requires` for resource usage and `parallel: true` when the runtime can safely spawn multiple workers.
4. **Add gateways for coordination** – When you need synchronization, splitting, or merging that is not covered by task IO alone, use a `gateway`. The compiler will expand it into the necessary Petri net plumbing.
5. **Validate by running `main_workflow.go`** – Point it at your YAML file to ensure it parses, compiles, and executes the resulting net.

By structuring workflows with this DSL you gain the full expressive power of Petri nets—natural resource constraints, deterministic synchronization, and analyzable execution—without giving up the readability of YAML-based orchestration.
