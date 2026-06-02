# reservation-service

[![Security](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)
[![Reliability](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)
[![Maintainability](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)
[![Duplications](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=coverage)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)

> **Purpose:** Core domain — manages spot inventory, reservation lifecycle, geofence validation, and no-show expiration.
> **Author:** Farid Triwicaksono

## Architecture Overview

![Architecture](docs/PintarParkir.architecture.svg)

## E2E Flow

![Flow Diagram](docs/flow.diagram.svg)

## Sequence Diagrams

- [Reservation Flow](docs/sequence-diagrams/01-reservation-flow.md)
- [Check-in Flow](docs/sequence-diagrams/02-checkin-flow.md)
- [Cancellation Flow](docs/sequence-diagrams/04-cancellation-flow.md)

## Tech Stack

- Go 1.25 + Gin (HTTP) + gRPC
- PostgreSQL (pgcrypto for PII encryption)
- Redis (caching + distributed locks)
- RabbitMQ (async event-driven via outbox pattern)
- Cloud Run (GCP) with auto-scaling
- OpenTelemetry (traces + metrics)

**Service-specific:** EXCLUDE constraint for double-book prevention, Redis distributed locks, geofence validation, outbox pattern

## API

See [OpenAPI Specification](docs/api-specifications/openapi-spec.yaml) and [AsyncAPI Specification](docs/api-specifications/asyncapi-spec.yaml).

## Running Locally

```bash
cp configs/.env.example configs/.env
make run
```

## Testing

```bash
make test          # unit tests
make test-coverage # with coverage report
```

## Deployment

CD via GitHub Actions → GCP Cloud Run (asia-southeast1).
Triggers on push to `main`.

Cloud Run URL: `https://reservation-service-725nddkmwq-as.a.run.app`
