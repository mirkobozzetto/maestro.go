# Maestro.go Implementation Plan

## Overview
Build a production-ready API orchestrator that coordinates polyglot services using gRPC as the unified communication protocol.

## Implementation Phases

### Phase 1: Project Foundation ✓
1. **Project Structure**
   - Create complete directory structure
   - Initialize Go module: `github.com/maestro/maestro.go`
   - Create CLAUDE.md with project instructions
   - Setup logging with zerolog

2. **Protocol Buffers**
   - Define maestro.proto with core services
   - Define common.proto with shared types
   - Setup protoc generation scripts

### Phase 2: Core Types & Parser
1. **Workflow Types** (internal/workflow/types.go)
   - Workflow struct with services registry
   - Step struct supporting sequential/parallel/conditional
   - Service definition with retry/timeout config
   - Input/Output handling with template support

2. **YAML Parser** (internal/workflow/parser.go)
   - Parse workflow YAML using gopkg.in/yaml.v3
   - Build DAG from steps
   - Template variable resolution using text/template
   - Basic validation (no cycles, valid references)

### Phase 3: Orchestration Engine
1. **Orchestrator Core** (internal/orchestrator/orchestrator.go)
   - Workflow execution context management
   - Service registry integration
   - Correlation ID generation
   - Structured logging for each step

2. **Step Executor** (internal/orchestrator/executor.go)
   - Sequential step execution
   - Parallel step execution using errgroup
   - Conditional step evaluation
   - Context propagation with timeout/cancellation

### Phase 4: gRPC Infrastructure
1. **Dynamic Client** (internal/grpc/client.go)
   - Use grpc.NewClient (2025 pattern)
   - Service discovery from workflow
   - Request/Response marshaling with protobuf
   - Metadata propagation (correlation ID, headers)

2. **Connection Pool** (internal/grpc/pool.go)
   - Round-robin connection distribution
   - Keepalive configuration
   - Health check integration
   - Connection lifecycle management

3. **Service Registry** (internal/grpc/registry.go)
   - In-memory service registration
   - Service endpoint resolution
   - Health status tracking
   - Circuit breaker per service

### Phase 5: Saga & Compensations
1. **Saga Coordinator** (internal/orchestrator/saga.go)
   - Transaction log for executed steps
   - Compensation tracking
   - Rollback orchestration (reverse order)
   - State persistence (in-memory for MVP)

2. **Compensation Executor**
   - Compensation method invocation
   - Error handling during rollback
   - Partial compensation support
   - Idempotent compensation checks

### Phase 6: Resilience Patterns
1. **Retry Logic**
   - Exponential backoff implementation
   - Configurable retry policies per service
   - Built-in gRPC retry configuration
   - Jitter for thundering herd prevention

2. **Circuit Breaker**
   - Use sony/gobreaker library
   - Per-service circuit configuration
   - State monitoring (closed/open/half-open)
   - Metrics collection

3. **Rate Limiting**
   - Token bucket per service
   - Configurable rates from workflow
   - Backpressure handling
   - Queue overflow management

### Phase 7: Adapters & Examples
1. **HTTP Adapter** (internal/adapters/http_adapter.go)
   - HTTP to gRPC bridge
   - JSON to Protobuf conversion
   - Header mapping
   - Response transformation

2. **Example Services**
   - Python/FastAPI adapter with gRPC wrapper
   - Node.js/Express adapter implementation
   - Native Go gRPC service example
   - Docker setup for each service

3. **Example Workflows**
   - user_onboarding.yaml with all features
   - order_processing.yaml with saga pattern
   - Simple hello_world.yaml for testing

### Phase 8: CLI & Operations
1. **CLI Tool** (cmd/maestro/main.go)
   - `serve` command to start orchestrator
   - `execute` command to run workflows
   - `status` command for workflow status
   - `validate` command for workflow validation

2. **Makefile**
   - Proto generation targets
   - Build and test commands
   - Docker compose helpers
   - Development shortcuts

3. **Docker Compose**
   - Orchestrator service
   - Example polyglot services
   - Network configuration
   - Volume mounts for workflows

### Phase 9: Testing & Validation
1. **Unit Tests**
   - Parser tests with valid/invalid YAML
   - Executor tests with mocked services
   - Saga compensation tests
   - Circuit breaker behavior tests

2. **Integration Tests**
   - End-to-end workflow execution
   - Service failure scenarios
   - Compensation verification
   - Timeout and cancellation tests

## Technical Decisions

### Dependencies
- **gRPC**: google.golang.org/grpc v1.64+
- **Protobuf**: google.golang.org/protobuf v1.34+
- **YAML**: gopkg.in/yaml.v3
- **Logging**: github.com/rs/zerolog
- **Circuit Breaker**: github.com/sony/gobreaker
- **Error Handling**: github.com/pkg/errors
- **Testing**: github.com/stretchr/testify

### Design Patterns
- **Repository Pattern**: For service registry
- **Factory Pattern**: For creating service clients
- **Strategy Pattern**: For retry policies
- **Observer Pattern**: For workflow events
- **Command Pattern**: For step execution

### Error Handling Strategy
- Wrap errors with context using fmt.Errorf("%w")
- Use custom error types for recoverable errors
- Implement error classification (transient/permanent)
- Structured error logging with correlation IDs

### Concurrency Model
- One goroutine per workflow execution
- Worker pool for parallel steps (10 workers default)
- Context-based cancellation propagation
- Mutex protection for shared state

## Success Criteria
1. ✅ Can parse and execute YAML workflows
2. ✅ Supports sequential, parallel, and conditional steps
3. ✅ Implements retry with exponential backoff
4. ✅ Handles compensation on failure
5. ✅ Provides structured logging with correlation IDs
6. ✅ Includes working examples for Python/Node/Go services
7. ✅ Passes all unit and integration tests
8. ✅ Builds and runs with single make command

## Risk Mitigation
- **Complexity**: Start with minimal features, iterate
- **Performance**: Profile and benchmark critical paths
- **Debugging**: Extensive logging and tracing
- **Testing**: High coverage from the start
- **Documentation**: Document as we build

## Next Steps
1. Create project structure
2. Implement basic types and parser
3. Build minimal executor
4. Add gRPC client
5. Iterate with testing at each step