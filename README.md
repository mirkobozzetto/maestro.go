# Maestro.go - Polyglot API Orchestrator

## Overview

Maestro.go is a powerful orchestrator that coordinates API services written in different languages (Python, Node.js, Go) using gRPC as the unified communication protocol. It executes workflows defined in YAML, handling retries, compensations, and distributed transactions automatically.

## Features

- 🔄 **Workflow Orchestration**: Execute complex workflows with sequential, parallel, and conditional steps
- 🛡️ **Saga Pattern**: Automatic compensation and rollback for distributed transactions
- 🔁 **Retry Logic**: Configurable exponential backoff for transient failures
- 🔌 **Circuit Breaker**: Fault tolerance with automatic service health management
- 🌐 **Polyglot Support**: Integrate services written in any language via gRPC
- 📊 **Structured Logging**: Detailed execution logs with correlation IDs
- ⚡ **Connection Pooling**: Efficient resource management for high throughput

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/maestro/maestro.go
cd maestro.go

# Install dependencies
go mod download

# Build the project
make build
```

### Running a Workflow

```bash
# Execute a workflow
make run WORKFLOW=examples/workflows/user_onboarding.yaml

# Or directly with the binary
./bin/maestro execute examples/workflows/user_onboarding.yaml \
  --input '{"email":"user@example.com","name":"John Doe"}'
```

## Workflow Definition

Workflows are defined in YAML with a simple, declarative syntax:

```yaml
name: user_onboarding
version: "1.0"
timeout: 30s

services:
  auth:
    type: grpc
    endpoint: "auth-service:50051"
    retry:
      attempts: 3
      backoff: exponential

steps:
  - id: create_user
    service: auth
    method: CreateUser
    input:
      email: "{{ .input.email }}"
    output: user
    compensate:
      method: DeleteUser
      input:
        id: "{{ .user.id }}"
```

## Architecture

```
[Client] → [Maestro Orchestrator] → [gRPC] → [Services]
                    ↓
              [Workflow Engine]
                    ↓
              [YAML Workflows]
```

## Development

### Available Commands

```bash
make help          # Show all available commands
make test          # Run tests
make lint          # Run linters
make proto         # Generate protobuf files
make docker-up     # Start development services
```

### Project Structure

```
maestro.go/
├── cmd/maestro/        # CLI application
├── internal/
│   ├── orchestrator/   # Core orchestration engine
│   ├── workflow/       # Workflow parsing and types
│   ├── grpc/          # gRPC client and registry
│   └── adapters/      # Service adapters
├── proto/             # Protocol buffer definitions
├── examples/          # Example workflows and services
└── Makefile          # Build automation
```

## Contributing

Contributions are welcome! Please read our contributing guidelines and submit pull requests to our repository.

## License

MIT License - see LICENSE file for details
