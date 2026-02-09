# Maestro.go

Glue your services into workflows that survive failure.

## The Problem

Modern backends aren't written in one language. You use Go or Rust where you need raw performance. Python when you need machine learning libraries or quick prototyping with FastAPI. Node.js for async I/O and the npm ecosystem. Java when enterprise libraries are the only option. Each service does what it's good at.

But these services need to work together. A request comes in and triggers a chain of operations across 3, 5, 10 different services — some in sequence, some in parallel. The output of one feeds into the next. When something breaks halfway through, the services that already ran need to clean up after themselves.

That's where the glue code starts. You write `if err != nil` chains, retry loops, timeout handling, rollback logic. It works for 2 services. At 5 services with parallel branches and conditional logic, it becomes the most fragile part of your stack — the part nobody wants to touch.

## What "Glue" Actually Means

Maestro sits between your services. It doesn't care what language they're written in, what framework they use, or where they run. It talks to them over gRPC or HTTP — the two protocols everything already speaks.

You describe the workflow in YAML:

- Which services to call, in what order
- Which steps run in parallel
- Which steps depend on the output of previous steps
- What to do when a step fails (undo logic per step)
- When to skip a step based on conditions

Maestro handles the rest: connection management, retries with backoff, circuit breaking when a service is down, timeout enforcement, and running undo operations in reverse order when something breaks.

Your services stay independent. They don't know about each other. They don't know they're part of a workflow. They just expose an API and respond to calls.

## Why Not Just Use [X]?

Every orchestration tool solves the same fundamental problem: making multiple services work together reliably. They differ in what they ask from you in return.

**Maestro** — _"Glue your services into workflows that survive failure"_
No cluster. No database. No SDK to integrate into your services. A single Go binary that reads a YAML file and coordinates your existing services over gRPC and HTTP. Your services don't need to know Maestro exists — they just need an API.

**Temporal** — _"What if your code never failed?"_
The most powerful option. Workflows survive server restarts, run for months, and recover from any failure. But you need to deploy a Temporal cluster with a database (Cassandra or PostgreSQL), learn their SDK, and write your workflows in code (Go, Java, Python, TypeScript). It's the right choice for mission-critical, long-running operations. It's overkill when your workflow takes 10 seconds.

**Orkes Conductor** — _"Orchestrate across any cloud, any language, any framework"_
Netflix-born, battle-tested at scale. JSON-based workflow definitions, visual editor, built-in observability. Requires a server deployment and comes with its own learning curve. Great for teams that need dashboards and governance.

**Kestra** — _"Language-agnostic orchestration platform"_
YAML-based like Maestro, with 600+ plugins and a visual editor. Needs a server and a database. Oriented toward data pipelines and scheduled jobs more than real-time service orchestration.

The tradeoff is clear: Maestro won't survive a server restart. There's no persistent state, no visual dashboard, no plugin ecosystem. If your workflows run for seconds or minutes and you need something that works without ops overhead, Maestro is for that. If you need workflows that run for days and must survive infrastructure failures, use Temporal.

## What a Workflow Looks Like

A workflow connects services that can be written in any language, running anywhere:

```yaml
name: my_workflow
version: "1.0"
timeout: 30s

services:
  fast_service:
    type: grpc
    endpoint: "perf-service:50051" # Could be Go, Rust, C++
    timeout: 5s
  ml_service:
    type: grpc
    endpoint: "ml-service:50052" # Could be Python, Julia
    timeout: 10s
  api_service:
    type: http
    endpoint: "http://api-service:3000" # Could be Node.js, Ruby, PHP
    retry:
      attempts: 3
      backoff: exponential
  legacy_service:
    type: http
    endpoint: "http://legacy:8080" # Could be Java, .NET
    timeout: 15s

steps:
  - id: step_1
    service: fast_service
    method: Process
    input:
      data: "{{ .input.payload }}"
    output: processed

  - parallel:
      - id: step_2a
        service: ml_service
        method: Analyze
        input:
          content: "{{ .processed.result }}"
        output: analysis
      - id: step_2b
        service: api_service
        method: "POST /enrich"
        input:
          id: "{{ .processed.id }}"
        output: enriched

  - id: step_3
    when: "{{ .analysis.confidence > 0.8 }}"
    service: legacy_service
    method: "POST /submit"
    input:
      analysis: "{{ .analysis }}"
      enrichment: "{{ .enriched }}"
    output: result
    compensate:
      method: "POST /cancel"
      input:
        submission_id: "{{ .result.id }}"

output:
  id: "{{ .result.id }}"
  status: "{{ .result.status }}"
```

What this does:

1. Calls a high-performance service (step 1)
2. In parallel: sends the result to an ML service AND an enrichment API (step 2a + 2b)
3. Conditionally submits to a legacy system if confidence is high enough (step 3)
4. If step 3 fails, its undo logic runs automatically

The services can be Go, Rust, Python, Node.js, Java, or anything else — Maestro doesn't care. It just needs a gRPC or HTTP endpoint.

## How It Handles Failure

Each step can define what "undo" means for itself. When step 3 fails, Maestro runs the undo logic of step 2, then step 1. In order. Automatically.

```
Step 1 (ok) → Step 2 (ok) → Step 3 (FAILS)
                                    ↓
                  Undo Step 2 ← Undo Step 1
```

Every undo has access to the data produced by all previous steps — so it can reference the exact IDs, paths, or tokens created earlier.

## What Keeps Services From Taking Everything Down

**Retries** — flaky service? Maestro retries with increasing delays. Permanent error? It stops immediately.

**Circuit breaker** — service keeps failing? Maestro stops calling it instead of making things worse. Checks again after 30 seconds.

**Connection pooling** — 5 persistent gRPC connections per service, no handshake overhead on every call.

**Timeouts** — per workflow, per service. Nothing hangs forever.

## Quick Start

```bash
make build

./bin/maestro validate workflow.yaml

./bin/maestro execute workflow.yaml \
  --input '{"payload":"your data here"}'
```

## How It Compares

|                   | Maestro | Temporal     | Conductor   | Kestra      |
| ----------------- | ------- | ------------ | ----------- | ----------- |
| How you define it | YAML    | Code         | JSON/Code   | YAML        |
| What you deploy   | Nothing | Cluster + DB | Server + DB | Server + DB |
| Language-agnostic | Yes     | Yes          | Yes         | Yes         |
| Failure handling  | Yes     | Yes          | Yes         | Partial     |
| Survives restarts | No      | Yes          | Yes         | Yes         |
| Time to first run | Minutes | Days         | Hours       | Hours       |

## Architecture

```
cmd/maestro/          CLI entry point
internal/
  application/        Orchestration engine, YAML parser, saga coordinator
    executor/         Step execution with worker pool
  domain/             Workflow, Step, Service models
  infrastructure/
    grpc/             Client, connection pool, circuit breaker, registry
    http/             HTTP adapter
pkg/proto/            Protobuf definitions
examples/workflows/   Ready-to-use workflow examples
```

## Development

```bash
make proto            # Generate protobuf code
make build            # Build binary
make test             # Run tests
make bench            # Run benchmarks
make docker-up        # Start with Docker
```

## License

MIT
