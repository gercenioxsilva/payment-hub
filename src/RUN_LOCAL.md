# Execução local (passo a passo)

## 1) Pré-requisitos
- Go 1.22+
- Docker + Docker Compose
- (Opcional) VS Code

## 2) Setup inicial
Crie o arquivo `.env`:
- Linux/macOS/WSL: `cp .env.example .env`
- Windows PowerShell: `Copy-Item .env.example .env`

Suba infraestrutura e prepare banco + filas:
```bash
make run-all
```

## 3) Rodar os serviços (4 terminais)
```bash
make run-ingress
make run-webhook
make run-publisher
make run-worker
```

## 4) UIs locais
- Swagger UI: http://localhost:8089
- ElasticMQ UI (SQS local): http://localhost:9325
- Adminer (DB): http://localhost:8088

## 5) Fluxos de teste
### PIX (async)
1) `POST /pix/payments` -> 202 com brCode/qr placeholder
2) Worker marca como PENDING
3) `POST /webhooks/pix` -> PAID/FAILED

### Cartão (crédito/débito)
1) `POST /card/*/payments` -> 202 (auth simulated)
2) Para simular confirmação final, use `POST /webhooks/card`

## 6) Observações (POC)
- SQS é emulado localmente com ElasticMQ (compatível com AWS SDK v2).
- DLQ/retry/circuit-breaker/bulkhead são implementados no worker (veja `apps/orchestrator-worker`).
