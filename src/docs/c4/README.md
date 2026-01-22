How to view the C4 model

- The diagrams are defined in Structurizr DSL at `docs/c4/workspace.dsl`.
- Options to render:
  - Use https://structurizr.com/dsl to paste the DSL and view diagrams in the browser.
  - Use the Structurizr CLI (`structurizr-cli`) to export to PlantUML or images.

Layers
- System Context: high-level view of the Payment Hub and external systems.
- Container View: main containers (Ingress API, Orchestrator Worker, Outbox Publisher, Webhook Receiver, PostgreSQL, SQS) and their protocols/libraries.

Technologies
- Go (chi, validator, oklog/ulid)
- AWS SDK for Go v2 (SQS)
- pgx (PostgreSQL driver)
- gobreaker (circuit breaker), cenkalti/backoff (retry)

