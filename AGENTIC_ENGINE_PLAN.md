# Petri Net Agentic Engine – Implementation Plan

This plan turns the current demo into a production-capable agentic workflow engine. Steps are ordered to de-risk correctness first, then add features, then harden for reliability.

## Phase 1 – Core Correctness (done)

- **Transactional token handling**: Transition firing is atomic with deterministic locking, rollback on failures, and guard rejections treated as “not ready”.
- **Output backpressure safety**: Output capacity checks account for tokens that are consumed and re-emitted (e.g., context/resource tokens).
- **Gateway semantics**: Barriers use `wait_for` fallback and task `<id>_done` signals; completion place is emitted.
- **DSL validation**: Duplicate/missing IDs and bad references fail before compile.
- **Context places**: DSL `contexts` and task `context:` binding wire a dedicated capacity-1 place for shared state.

## Phase 2 – Task Runtime & Scheduling

- **Action registry**: Map DSL `task.type` to implementations (HTTP, LLM, shell, aggregator, splitter). Inject dependencies (HTTP client, LLM client) via constructors to keep clean boundaries.
- **Execution policies**: Add per-task timeout, retry/backoff, idempotency flag, and max concurrency. Respect resource tokens plus a scheduler-level limit.
- **Guards & conditions**: Support typed guard functions (e.g., expressions) and branch routing (conditional splits) to enable agent decisions.

## Phase 3 – Persistence & Observability

- **State persistence**: Durable storage for places, transitions, and tokens (e.g., BoltDB/SQLite) with checkpoint/replay so long-running agent workflows survive restarts.
- **Event log**: Append-only log of transition firings and token movements for auditing and debugging.
- **Metrics & tracing**: Emit counters for fired transitions, queue depths, failures, and latencies; optional OpenTelemetry spans around task actions.

## Phase 4 – Developer Experience & Testing

- **CLI/SDK**: Add commands to validate, inspect, and simulate DSL nets; expose a Go API for embedding and a future REST layer.
- **Testing**: Property tests for place/transition invariants; integration tests for gateways; golden-file tests for DSL validation errors; concurrency tests for race safety.
- **Examples**: Expand DSL examples to cover barriers, conditional routing, retries, and resource contention.

## Acceptance Criteria

- No token loss or duplication under concurrent firing; backpressure cannot drop data.
- DSL files with typos or missing references fail validation with actionable errors.
- Gateways enforce synchronization/fan-out/fan-in semantics in execution.
- Tasks run through a registry with timeouts and retries; engine exposes metrics and logs; workflows can resume after restart from persisted state.
