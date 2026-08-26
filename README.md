# ModularBuild Site Operations

ModularBuild coordinates prefabricated building modules from the assembly partner to the construction site. A fabricator registers a module movement, a site planner assigns it to a time-bounded lift window, quality staff release the module, and the installation crew records site-safety clearance before lifting and installation. The workflow connects planning, transport capacity, quality evidence, safety approval, installation events, audit history, and durable outbox delivery.

The service is intentionally a backend operations system rather than an inventory or booking product. A movement cannot advance through loading or departure until the associated quality and site-safety checks are released. Capacity reservations, optimistic versions, idempotency keys, tenant filtering, and audit writes are enforced in the persistence layer.

## Runtime

The service uses Go 1.22, PostgreSQL 16, `pgx/v5`, Chi, and bcrypt. It provides server-side revocable sessions, request IDs, structured logs, panic recovery, liveness/readiness endpoints, graceful shutdown, and an outbox worker with retry and stale-claim recovery.

Start the dependency and service locally:

```bash
docker compose up -d postgres
set -a; source .env.example; set +a
GOTOOLCHAIN=local go run ./cmd/server
```

The default service address is `127.0.0.1:8080`; the test database is exposed on `127.0.0.1:55433`.

## Business API

- `POST /v1/auth/login` and `POST /v1/auth/logout` manage expiring, revocable sessions.
- `POST /v1/module-moves` registers a movement; `GET /v1/module-moves` supports tenant-scoped status pagination.
- `POST /v1/lift-windows` creates a site lift window and `/open` makes it available for assignment.
- `POST /v1/module-moves/{id}/assign` atomically reserves lift capacity and advances the movement.
- `POST /v1/quality-cases` records quality review transitions.
- `POST /v1/site-safety-checks` records installation-site safety decisions.
- `POST /v1/module-moves/{id}/transition` performs guarded loading, departure, and installation transitions.
- `GET /v1/operations/summary` returns a role-protected operational summary.

Roles are `fabricator`, `site_planner`, and `installation_crew`. Each request carries a tenant boundary through the service and repository layers.

## Persistence and recovery

Migrations are embedded from `migrations/*.sql` and recorded in `schema_migrations`. The schema contains tenants, users, sessions, module movements, lift windows, quality cases, site-safety checks, handling events, audit events, idempotency records, and outbox events. Migration startup is repeatable and a failed migration is rolled back.

The assignment path uses one transaction for movement state, lift capacity, handling/audit data, and the outbox event. PostgreSQL row locks and version predicates prevent concurrent over-allocation. Exact idempotent requests return their persisted result, while changed replays are rejected. The worker claims outbox records with leases, retries transient failures, records permanent failures, and recovers expired claims after restart.

## Verification

With PostgreSQL running:

```bash
GOTOOLCHAIN=local go test ./... -count=1
GOTOOLCHAIN=local go test -race ./... -count=1
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./...
```

The tests cover domain state transitions, role authorization, migrations and reopen recovery, transaction rollback, capacity conflicts, idempotency, pagination, HTTP error mapping, tenant isolation, worker retry/cancellation, and the complete movement-to-installation lifecycle.

## Container

Build the production image with `docker build -t modularbuild:local .`. Provide `DATABASE_URL` at runtime. The image starts `./cmd/server` and does not contain development credentials.
