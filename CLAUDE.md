# Maestro.go - Instructions for Claude Code

## Project Overview
Maestro.go is a polyglot API orchestrator that coordinates services written in different languages (Python, Node.js, Go) using gRPC as the unified communication protocol.

## Development Guidelines

### Code Style
- Go 1.21+ idiomatic code
- No premature interfaces - only when needed
- Clear variable names over comments
- Wrap errors with context: `fmt.Errorf("%w", err)`
- Use zerolog for structured logging

### Testing Requirements
- Write tests alongside implementation
- Target 80% code coverage
- Use table-driven tests
- Mock external dependencies

### Workflow Format
```yaml
name: workflow_name
version: "1.0"
timeout: 30s

services:
  service_name:
    type: grpc
    endpoint: "host:port"
    retry:
      attempts: 3
      backoff: exponential

steps:
  - id: step_id
    service: service_name
    method: MethodName
    input:
      field: "{{ .variable }}"
    output: result_name
    compensate:
      method: CompensateMethod
      input:
        id: "{{ .result_name.id }}"
```

### Implementation Priorities
1. Correctness over performance
2. Simplicity over complexity
3. Explicit over implicit
4. Testability from the start

### Common Commands
```bash
# Generate protobuf
make proto

# Run tests
make test

# Build
make build

# Run workflow
make run WORKFLOW=examples/workflows/user_onboarding.yaml
```

### Directory Structure
- `cmd/maestro/` - CLI application
- `internal/orchestrator/` - Core orchestration engine
- `internal/workflow/` - Workflow parsing and types
- `internal/grpc/` - gRPC client and registry
- `internal/adapters/` - Service adapters
- `proto/` - Protocol buffer definitions
- `examples/` - Example workflows and services

### Key Patterns
- Saga pattern for distributed transactions
- Circuit breaker for fault tolerance
- Exponential backoff for retries
- Compensation for rollback

### Dependencies
- google.golang.org/grpc - gRPC framework
- google.golang.org/protobuf - Protocol buffers
- gopkg.in/yaml.v3 - YAML parsing
- github.com/rs/zerolog - Structured logging
- github.com/sony/gobreaker - Circuit breaker

### Development Workflow
1. Make changes
2. Run `make test`
3. Run `make lint` (when available)
4. Test with example workflows
5. Update documentation if needed