Project Context for AI Contributors

Scope
- Service: Payment Hub POC (Go)
- Binaries:
  - `apps/ingress-api/cmd`: HTTP ingress for client requests
  - `apps/orchestrator-worker/cmd`: SQS worker to orchestrate providers
  - `apps/outbox-publisher/cmd`: publishes outbox (not covered here)
  - `apps/webhook-receiver/cmd`: receives provider callbacks
  - `scripts/migrate`: run DB migrations
  - `scripts/queue-create`: create SQS queues (main and DLQ)

Key Concepts
- Idempotency: All create endpoints require `Idempotency-Key` header. Repository enforces unique `(order_id, idempotency_key)`.
- Outbox Pattern: `Repo.CreatePaymentWithOutbox` writes a payment_intent and an outbox event `PaymentRequested`. Worker consumes and calls providers.
- Providers: Located under `internal/providers`. Use `Provider` interface and `Authorize` method. Mock providers: `PixMock`, `CardMock`.
- Policy Routing: `config/policy.yml` defines routing and fallback chains per method.

Data Model
- `internal/domain/payment.go`: `PaymentIntent` with fields for PIX, card, and status tracking. Statuses: CREATED, QUEUED, PROCESSING, PENDING, PAID, FAILED. Methods include `PIX` and card variants.
- DB schema in `internal/persistence/migrations/*.sql`.

HTTP API
- Defined in `openapi.yaml`. Notable routes:
  - `POST /payments/init`: generic initializer that accepts extended schema (customer, vendor, items, paymentData, etc.).
    - Request: `StartPaymentRequest` (extended fields).
    - Response (HTTP 200): StartPaymentResponseV2 with shape: `status`, `methods[]` (method-level status), `paymentKey`, `partnerUniqueId`, `code`, nested `message`, `operationId`.
    - Internals: Creates QUEUED intent and enqueues outbox event.
  - `POST /pix/payments`, `POST /card/*/payments`: specialized endpoints.
  - `GET /payments/{paymentId}`: fetches intent status.

Implementation Notes
- Ingress handlers live in `apps/ingress-api/cmd/main.go`.
- Use `ulid` for `PaymentID`.
- Amount handling: specialized endpoints use cents; generic initializer maps `paymentData.amount` (float) to `AmountCents` (int64) by multiplying by 100.
- Response mapping: generic initializer returns V2 envelope described above. For PIX, optional `pixInfo` is present (fields empty in POC).
- Currency: default to "BRL" unless specified otherwise.
- When extending request models, add JSON tags and keep validation minimal on generic initializer; specialized endpoints use `go-playground/validator`.

Worker
- File: `apps/orchestrator-worker/cmd/main.go`.
- Reads SQS, parses `OutboxEvent`, fetches `PaymentIntent`, selects provider chain from policy, applies bulkheads and circuit breakers, and retries with exponential backoff.
- On success, persists provider fields via `Repo.UpdateProviderFields`.

Local Dev
- Config from `internal/config` and `.env`/`developer.env`.
- Start ingress with `go run ./apps/ingress-api/cmd`.
- Run migrations with `go run ./scripts/migrate`.
- Create queues with `go run ./scripts/queue-create`.

Conventions
- Keep changes minimal and consistent with existing style.
- Always update `openapi.yaml` when adding endpoints.
- Use idempotency checks for create-like operations.
