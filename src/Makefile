SHELL := /bin/bash
.PHONY: infra-up infra-down migrate queue run-ingress run-webhook run-publisher run-worker run-all

infra-up:
	docker-compose up -d

infra-down:
	docker-compose down

migrate:
	go run ./scripts/migrate.go

queue:
	go run ./scripts/queue_create.go

run-ingress:
	INGRESS_PORT=8080 go run ./apps/ingress-api/cmd

run-webhook:
	WEBHOOK_PORT=8081 go run ./apps/webhook-receiver/cmd

run-publisher:
	go run ./apps/outbox-publisher/cmd

run-worker:
	go run ./apps/orchestrator-worker/cmd

run-all:
	$(MAKE) infra-up migrate queue
	@echo "Now run each app in separate terminals (or use VS Code tasks)."
