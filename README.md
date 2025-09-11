# Maestro.go

Polyglot API orchestrator that coordinates services written in different languages using gRPC and HTTP.

## Features

- Workflow orchestration with YAML definitions
- Saga pattern for distributed transactions
- Automatic retry with exponential backoff
- Circuit breaker for fault tolerance
- Support for gRPC and HTTP services
- Template resolution for dynamic values
- Compensation for rollback operations

## Installation

```bash
# Install tools
make install-tools

# Build
make build

# Run tests
make test
```

## Quick Start

```bash
# Build
make build

# Start orchestrator server
./maestro serve --port 8080

# Validate a workflow
./maestro validate workflow.yaml

# Execute workflow with input file
./maestro execute workflow.yaml --input-file input.json
```

## Use Case Example

Multi-service user onboarding workflow:

```yaml
name: user_onboarding
version: "1.0"

services:
  auth_service:
    type: grpc
    endpoint: "auth-service:50051"

  billing_service:
    type: grpc
    endpoint: "billing-service:50052"

  crm_service:
    type: http
    endpoint: "http://crm-api:8080"

  email_service:
    type: grpc
    endpoint: "email-service:50053"

steps:
  - id: create_user
    service: auth_service
    method: CreateUser
    input:
      email: "{{ .input.email }}"
      password: "{{ .input.password }}"
    output: user

  - id: setup_subscription
    service: billing_service
    method: CreateSubscription
    input:
      user_id: "{{ .user.id }}"
      plan: "{{ .input.plan }}"
    output: subscription
    compensate:
      method: CancelSubscription
      input:
        subscription_id: "{{ .subscription.id }}"

  - id: add_to_crm
    service: crm_service
    method: POST /contacts
    input:
      user_id: "{{ .user.id }}"
      email: "{{ .input.email }}"
      subscription_tier: "{{ .subscription.tier }}"
    output: crm_contact
    compensate:
      method: DELETE /contacts/{{ .crm_contact.id }}

  - id: send_welcome_email
    service: email_service
    method: SendTemplate
    input:
      to: "{{ .input.email }}"
      template: "welcome"
      vars:
        name: "{{ .input.name }}"
        plan: "{{ .subscription.plan_name }}"

output:
  user_id: "{{ .user.id }}"
  subscription_id: "{{ .subscription.id }}"
```

If any step fails, compensations run in reverse order to clean up.

## Architecture

```
cmd/maestro/          - CLI application
internal/
  application/        - Orchestration logic
  domain/            - Core domain models
  infrastructure/    - gRPC/HTTP adapters
pkg/proto/           - Protocol buffer definitions
```

## Development

```bash
# Generate protobuf
make proto

# Run with debug logging
./maestro execute workflow.yaml --debug

# Run benchmarks
make bench
```
