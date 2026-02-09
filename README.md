# Maestro.go

Glue your services into workflows that survive failure.

## The Problem

You have 5 services that need to work together to complete one business operation. You wrote the glue code yourself — `if err != nil` chains, retry loops, manual cleanup when something fails halfway.

It works. Until it doesn't. Your identity check calls 3 external APIs and one times out — now you have a half-verified user stuck in limbo. Your deployment script created the load balancer but crashed before setting up DNS — now you're cleaning up cloud resources by hand. Your video processing pipeline transcoded the file but the thumbnail service was down — now you have orphan files in S3.

You could deploy Temporal. But now you need a cluster, a database, and a team that understands it. For 5 services.

Maestro is a single Go binary. You describe the workflow, you run it. If a step fails, previous steps undo themselves. If a service is flaky, it retries. If a service is dead, it stops hammering it. No infrastructure to deploy.

## What People Actually Use This For

### Running Background Checks Across Multiple Sources

This is what Checkr does at scale with Temporal. When someone applies for a job, you need to hit a criminal records database, an identity verification service, a county court API, and a credit check — all in parallel because each one takes different amounts of time. You merge the results, score them, and make a decision. If the credit check provider is down, you still get results from the other three instead of failing the whole check.

```yaml
name: background_check
version: "1.0"
timeout: 30s

services:
  identity:
    type: http
    endpoint: "http://id-verify:3000"
    timeout: 10s
  criminal:
    type: grpc
    endpoint: "criminal-records:50051"
    timeout: 10s
  court:
    type: http
    endpoint: "http://court-api:3001"
    timeout: 15s
  credit:
    type: grpc
    endpoint: "credit-check:50052"
    timeout: 10s
  scoring:
    type: grpc
    endpoint: "scoring-engine:50053"

steps:
  - parallel:
      - id: verify_identity
        service: identity
        method: "POST /verify"
        input:
          ssn: "{{ .input.ssn }}"
          name: "{{ .input.name }}"
          dob: "{{ .input.dob }}"
        output: identity_result
      - id: check_criminal
        service: criminal
        method: SearchRecords
        input:
          name: "{{ .input.name }}"
          state: "{{ .input.state }}"
        output: criminal_result
      - id: check_courts
        service: court
        method: "POST /search"
        input:
          name: "{{ .input.name }}"
          counties: "{{ .input.counties }}"
        output: court_result
      - id: check_credit
        service: credit
        method: GetScore
        input:
          ssn: "{{ .input.ssn }}"
        output: credit_result

  - id: score
    service: scoring
    method: EvaluateCandidate
    input:
      identity: "{{ .identity_result }}"
      criminal: "{{ .criminal_result }}"
      courts: "{{ .court_result }}"
      credit: "{{ .credit_result }}"
    output: decision

output:
  status: "{{ .decision.status }}"
  risk_level: "{{ .decision.risk_level }}"
  report_id: "{{ .decision.report_id }}"
```

### Processing Media Files Through Multiple Services

This is what VEED.IO handles millions of times a day. A user uploads a video. You need to validate the format, transcode it into multiple resolutions in parallel, generate thumbnails, and push the results to your CDN. If transcoding fails, you clean up the partial files — no orphans left on S3.

```yaml
name: video_processing
version: "1.0"
timeout: 5m

services:
  validator:
    type: grpc
    endpoint: "media-validator:50051"
  transcoder:
    type: grpc
    endpoint: "transcoder:50052"
    timeout: 3m
  thumbnailer:
    type: http
    endpoint: "http://thumbnailer:3000"
    timeout: 30s
  cdn:
    type: http
    endpoint: "http://cdn-api:3001"
    retry:
      attempts: 3
      backoff: exponential

steps:
  - id: validate
    service: validator
    method: ValidateMedia
    input:
      file_url: "{{ .input.file_url }}"
      expected_type: "video"
    output: validated

  - parallel:
      - id: transcode_720p
        service: transcoder
        method: Transcode
        input:
          source: "{{ .validated.storage_path }}"
          resolution: "720p"
          codec: "h264"
        output: video_720
        compensate:
          method: DeleteFile
          input:
            path: "{{ .video_720.output_path }}"
      - id: transcode_1080p
        service: transcoder
        method: Transcode
        input:
          source: "{{ .validated.storage_path }}"
          resolution: "1080p"
          codec: "h264"
        output: video_1080
        compensate:
          method: DeleteFile
          input:
            path: "{{ .video_1080.output_path }}"
      - id: generate_thumbnails
        service: thumbnailer
        method: "POST /generate"
        input:
          source: "{{ .validated.storage_path }}"
          count: 3
          timestamps: "{{ .validated.suggested_timestamps }}"
        output: thumbnails

  - id: publish
    service: cdn
    method: "POST /publish"
    input:
      assets:
        - "{{ .video_720.output_path }}"
        - "{{ .video_1080.output_path }}"
      thumbnails: "{{ .thumbnails.paths }}"
      metadata:
        duration: "{{ .validated.duration }}"
        title: "{{ .input.title }}"
    output: published

output:
  playback_url: "{{ .published.playback_url }}"
  thumbnail_url: "{{ .published.thumbnail_url }}"
```

### Deploying Infrastructure Without Leaving a Mess

This is what Chronosphere built with Temporal for their deployment system. You need to create cloud resources in a specific order — provision the instance, configure networking, deploy your app, run health checks, then update the load balancer. If health checks fail after the instance is running, everything tears down in reverse. No zombie instances burning money.

```yaml
name: deploy_service
version: "1.0"
timeout: 10m

services:
  cloud:
    type: grpc
    endpoint: "cloud-provisioner:50051"
    timeout: 2m
  deployer:
    type: grpc
    endpoint: "deployer:50052"
    timeout: 3m
  health:
    type: http
    endpoint: "http://health-checker:3000"
    timeout: 60s
  loadbalancer:
    type: grpc
    endpoint: "lb-manager:50053"
  notifications:
    type: http
    endpoint: "http://notifications:3001"

steps:
  - id: provision
    service: cloud
    method: CreateInstance
    input:
      image: "{{ .input.image }}"
      size: "{{ .input.instance_size }}"
      region: "{{ .input.region }}"
    output: instance
    compensate:
      method: DestroyInstance
      input:
        instance_id: "{{ .instance.id }}"

  - id: deploy_app
    service: deployer
    method: Deploy
    input:
      instance_id: "{{ .instance.id }}"
      artifact: "{{ .input.artifact_url }}"
      env: "{{ .input.environment }}"
    output: deployment
    compensate:
      method: Undeploy
      input:
        deployment_id: "{{ .deployment.id }}"

  - id: check_health
    service: health
    method: "POST /check"
    input:
      target: "{{ .instance.ip }}"
      port: "{{ .deployment.port }}"
      expected_status: 200

  - id: register_lb
    service: loadbalancer
    method: RegisterTarget
    input:
      lb_id: "{{ .input.lb_id }}"
      target_ip: "{{ .instance.ip }}"
      target_port: "{{ .deployment.port }}"
    output: registration
    compensate:
      method: DeregisterTarget
      input:
        registration_id: "{{ .registration.id }}"

  - id: notify
    service: notifications
    method: "POST /send"
    input:
      channel: "{{ .input.slack_channel }}"
      message: "Deployed {{ .input.artifact_url }} to {{ .input.region }}"
```

Health check fails → app is undeployed, instance is destroyed. No orphan resources.

### Enriching Leads From Multiple Data Sources

This is what Cargo built as a revenue orchestration platform. A new lead comes in from your website. You need to enrich it — hit Clearbit for company data, your internal DB for existing interactions, a scoring service to rank the lead — all in parallel. Then route it to the right sales rep based on territory and deal size.

```yaml
name: lead_enrichment
version: "1.0"
timeout: 15s

services:
  clearbit:
    type: http
    endpoint: "http://clearbit-proxy:3000"
    timeout: 5s
  internal_db:
    type: grpc
    endpoint: "crm-service:50051"
    timeout: 3s
  scoring:
    type: grpc
    endpoint: "lead-scoring:50052"
  routing:
    type: grpc
    endpoint: "lead-routing:50053"
  crm:
    type: http
    endpoint: "http://crm-api:3001"
  notifications:
    type: http
    endpoint: "http://notifications:3002"

steps:
  - parallel:
      - id: enrich_company
        service: clearbit
        method: "GET /companies/{{ .input.domain }}"
        output: company
      - id: check_existing
        service: internal_db
        method: FindContactHistory
        input:
          email: "{{ .input.email }}"
        output: history

  - id: score_lead
    service: scoring
    method: Score
    input:
      company_size: "{{ .company.employees }}"
      industry: "{{ .company.industry }}"
      previous_interactions: "{{ .history.count }}"
      source: "{{ .input.source }}"
    output: scored

  - id: route
    service: routing
    method: AssignRep
    input:
      territory: "{{ .company.country }}"
      deal_size: "{{ .scored.estimated_value }}"
      score: "{{ .scored.score }}"
    output: assignment

  - parallel:
      - id: update_crm
        service: crm
        method: "POST /leads"
        input:
          email: "{{ .input.email }}"
          company: "{{ .company }}"
          score: "{{ .scored.score }}"
          assigned_to: "{{ .assignment.rep_id }}"
      - id: notify_rep
        service: notifications
        method: "POST /send"
        input:
          to: "{{ .assignment.rep_email }}"
          template: "new_lead"
          data:
            company: "{{ .company.name }}"
            score: "{{ .scored.score }}"
```

## How It Handles Failure

Each step can define what "undo" means for itself. When step 3 fails, Maestro runs the undo logic of step 2, then step 1. In order. Automatically.

```
Step 1 (ok) → Step 2 (ok) → Step 3 (FAILS)
                                    ↓
                  Undo Step 2 ← Undo Step 1
```

Every undo has access to the data produced by all previous steps — so it can reference the exact instance ID to destroy, the exact file path to delete, the exact registration to remove.

## What Keeps Services From Taking Everything Down

**Retries** — flaky service? Maestro retries with increasing delays. Permanent error? It stops immediately.

**Circuit breaker** — service keeps failing? Maestro stops calling it instead of making things worse. Checks again after 30 seconds.

**Connection pooling** — 5 persistent connections per service, no handshake overhead on every call.

**Timeouts** — per workflow, per service. Nothing hangs forever.

## Quick Start

```bash
make build

./bin/maestro validate examples/workflows/order_processing.yaml

./bin/maestro execute examples/workflows/user_onboarding.yaml \
  --input '{"email":"john@example.com","name":"John","plan":"premium"}'
```

## How It Compares

|                    | Maestro              | Temporal              | Conductor   | Airflow        |
| ------------------ | -------------------- | --------------------- | ----------- | -------------- |
| Definition         | YAML                 | Code (Go/Java/Python) | JSON/Code   | Python DAGs    |
| Infrastructure     | None (single binary) | Cluster + DB          | Server + DB | Scheduler + DB |
| Failure handling   | Yes                  | Yes                   | Yes         | No             |
| Parallel execution | Yes                  | Yes                   | Yes         | Yes            |
| Survives restarts  | No                   | Yes                   | Yes         | Yes            |
| Time to first run  | Minutes              | Days                  | Hours       | Hours          |

Maestro won't survive a server restart like Temporal does. That's the tradeoff. If your workflows take seconds or minutes and you need something that just works without ops overhead, Maestro is for that. If you need workflows that run for days and must survive infrastructure failures, use Temporal.

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
