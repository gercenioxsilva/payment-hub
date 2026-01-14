CREATE TABLE IF NOT EXISTS payment_intent (
  payment_id       TEXT PRIMARY KEY,
  order_id         TEXT NOT NULL,
  idempotency_key  TEXT NOT NULL,
  amount_cents     BIGINT NOT NULL,
  currency         TEXT NOT NULL,
  method           TEXT NOT NULL,
  status           TEXT NOT NULL,
  provider         TEXT,
  e2e_id           TEXT,
  txid             TEXT,
  payer_name       TEXT,
  payer_document   TEXT,
  payer_bank       TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(order_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS outbox (
  id              BIGSERIAL PRIMARY KEY,
  event_type      TEXT NOT NULL,
  payload_json    TEXT NOT NULL,
  published_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox(published_at) WHERE published_at IS NULL;
