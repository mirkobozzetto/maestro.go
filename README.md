# Maestro.go

Polyglot API orchestrator that coordinates services written in different languages using gRPC and HTTP.

## When to Use Maestro

**Not for simple workflows** - If you just need to chain 3 API calls, use a message queue (RabbitMQ, Redis).

Maestro is designed for **complex orchestration** where queues become a nightmare:

- **Multi-branch workflows** with conditional logic (if X then Y else Z)
- **Parallel execution** with synchronization (call 5 APIs simultaneously, wait for all)
- **Different compensation per step** (not just "undo", but specific rollback actions)
- **Stateful recovery** - Resume exactly where it crashed after failures
- **Business-readable workflows** - Non-developers can understand YAML definitions

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

## Complex Workflow Example

Bank loan processing with parallel checks and conditional approval:

```yaml
name: loan_processing
version: "1.0"

steps:
  - id: verify_identity
    service: kyc_service
    method: VerifyIdentity
    output: identity

  - parallel:
      - id: credit_check
        service: credit_bureau
        method: GetCreditScore
        output: credit_score
        
      - id: income_verification
        service: employer_api
        method: VerifyIncome
        output: income
        
      - id: fraud_check
        service: fraud_detection
        method: AnalyzeRisk
        output: fraud_risk

  - id: auto_approve
    when: "{{ .credit_score.value > 700 && .fraud_risk.level == 'low' }}"
    service: loan_service
    method: AutoApprove
    output: approval
    compensate:
      method: CancelApproval
      input:
        loan_id: "{{ .approval.loan_id }}"

  - id: manual_review
    when: "{{ .credit_score.value > 600 && .credit_score.value <= 700 }}"
    service: review_queue
    method: CreateReviewTask
    input:
      priority: "{{ .fraud_risk.level == 'high' ? 'urgent' : 'normal' }}"
    output: review_task

  - id: reject
    when: "{{ .credit_score.value <= 600 }}"
    service: loan_service
    method: RejectApplication
    output: rejection

  - id: generate_contract
    when: "{{ .approval.status == 'approved' }}"
    service: document_service
    method: GenerateContract
    input:
      terms: "{{ .approval.terms }}"
      rate: "{{ .approval.interest_rate }}"
    output: contract
```

This shows why Maestro exists: parallel execution, conditional branches, and specific compensations per step - impossible to manage with simple queues.

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

## Status

This project is under active development and evolves based on real-world usage. Feel free to:

- Open issues with questions or suggestions
- Submit pull requests
- Fork and adapt it to your needs
- Share your use cases

Your feedback helps shape the project. Don't hesitate to reach out!
