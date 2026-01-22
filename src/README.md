# ClickPay Go POC (PIX + SQS local (ElasticMQ) + Postgres)

## Requisitos
- Go 1.22+
- Docker + Docker Compose
- VS Code (opcional)

## Quickstart
1) Copie o env:
- Linux/macOS/WSL: `cp .env.example .env`
- Windows PowerShell: `Copy-Item .env.example .env`

2) Infra + migrations + fila:
`make run-all`

3) Rode as apps (em terminais separados) ou use as tasks do VS Code:
- `make run-ingress`
- `make run-webhook`
- `make run-publisher`
- `make run-worker`

## Teste PIX
Criar PIX (202):
```bash
curl -s -X POST http://localhost:8080/pix/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: idem-123' \
  -d '{
    "orderId":"order-1001",
    "amount":{"valueCents":1990,"currency":"BRL"},
    "payer":{"name":"Joao da Silva","document":"12345678909","bank":"ITA"},
    "pix":{"key":"email@merchant.com","payerMessage":"Pedido 1001","expiresInSeconds":900}
  }'
```

Buscar status:
```bash
curl -s http://localhost:8080/payments/<paymentId>
```

Simular webhook PAID:
```bash
curl -s -X POST http://localhost:8081/webhooks/pix \
  -H 'Content-Type: application/json' \
  -d '{"paymentId":"<paymentId>","status":"PAID","e2eId":"E123456789012345678901234567890","txid":"TX-ABCDEFGH"}'
```


## Swagger UI
Depois de `make infra-up`, acesse:
- Swagger UI: http://localhost:8089
- ElasticMQ UI: http://localhost:9325
- Adminer: http://localhost:8088 (System: PostgreSQL, Server: postgres, User/Pass/DB: clickpay)

## Arquitetura (mapeada ao diagrama)
- **Ingress API** (`apps/ingress-api`): expõe endpoints públicos, validação e idempotência.
- **Fila/Stream** (`apps/outbox-publisher` + ElasticMQ): publica eventos de pagamento.
- **Orchestrator Worker** (`apps/orchestrator-worker`): consome eventos e roteia para provedores (mock).
- **Webhook Receiver** (`apps/webhook-receiver`): recebe callbacks e normaliza status.
- **Payment Ledger** (Postgres): persistência do state machine de pagamentos.
- **Read Model**: consulta de status via `/payments/{paymentId}`.

## Cartão (crédito/débito)
### Crédito
```bash
curl -s -X POST http://localhost:8080/card/credit/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: idem-cc-1' \
  -d '{
    "orderId":"order-2001",
    "amount":{"valueCents":4990,"currency":"BRL"},
    "card":{"holderName":"Maria Silva","number":"4111111111111111","expMonth":12,"expYear":2030,"cvv":"123","brand":"VISA"}
  }'
```
### Débito
```bash
curl -s -X POST http://localhost:8080/card/debit/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: idem-db-1' \
  -d '{
    "orderId":"order-2002",
    "amount":{"valueCents":2590,"currency":"BRL"},
    "card":{"holderName":"Maria Silva","number":"5555555555554444","expMonth":12,"expYear":2030,"cvv":"123","brand":"MASTERCARD"}
  }'
```
Opcional: para simular settlement/capture via webhook:
```bash
curl -s -X POST http://localhost:8081/webhooks/card \
  -H 'Content-Type: application/json' \
  -d '{"paymentId":"<paymentId>","status":"PAID"}'
```

## Payment Init (V2 envelope, HTTP 200)
Exemplo de requisição (PIX) usando o payload completo (há um arquivo pronto em `docs/examples/payment-init.json`):
```bash
curl -s -X POST http://localhost:8080/payments/init \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: idem-init-1' \
  --data-binary @docs/examples/payment-init.json | jq
```

Resposta (200) no envelope V2:
```json
{
  "status": "Q",
  "methods": [
    {
      "methodType": "PIX",
      "status": "Q",
      "methodId": 1,
      "operationId": "<paymentId>",
      "methodKey": "<paymentId>",
      "message": {"source":0, "code":"OK", "message":"QUEUED", "info":""},
      "redirectInfo": {"url":""},
      "acquirer": [],
      "antifraud": [],
      "pixInfo": {"qRContent":"", "qrCopyPaste":"", "qRImage":""}
    }
  ],
  "paymentKey": "<paymentId>",
  "partnerUniqueId": "order-3001",
  "code": 200,
  "message": {"source":0, "code":"OK", "message":"QUEUED", "info":""},
  "operationId": "<paymentId>"
}
```
