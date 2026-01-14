package persistence

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/you/clickpay-go-poc/internal/domain"
)

type Repo struct{ DB *pgxpool.Pool }

func (r Repo) FindByPaymentID(ctx context.Context, paymentId string) (*domain.PaymentIntent, error) {
	row := r.DB.QueryRow(ctx, `
    SELECT payment_id, order_id, idempotency_key, amount_cents, currency, method, status, provider,
           e2e_id, txid, payer_name, payer_document, payer_bank, created_at, updated_at
      FROM payment_intent WHERE payment_id=$1`, paymentId)

	var p domain.PaymentIntent
	var method, status string
	err := row.Scan(&p.PaymentID, &p.OrderID, &p.IdempotencyKey, &p.AmountCents, &p.Currency, &method, &status, &p.Provider,
		&p.E2EID, &p.TXID, &p.PayerName, &p.PayerDocument, &p.PayerBank, &p.CreatedAt, &p.UpdatedAt)
	if err != nil { return nil, err }
	p.Method = domain.PaymentMethod(method)
	p.Status = domain.PaymentStatus(status)
	return &p, nil
}

func (r Repo) FindByOrderAndIdem(ctx context.Context, orderId, idem string) (*domain.PaymentIntent, error) {
	row := r.DB.QueryRow(ctx, `
    SELECT payment_id, order_id, idempotency_key, amount_cents, currency, method, status, provider,
           e2e_id, txid, payer_name, payer_document, payer_bank, created_at, updated_at
      FROM payment_intent WHERE order_id=$1 AND idempotency_key=$2`, orderId, idem)

	var p domain.PaymentIntent
	var method, status string
	err := row.Scan(&p.PaymentID, &p.OrderID, &p.IdempotencyKey, &p.AmountCents, &p.Currency, &method, &status, &p.Provider,
		&p.E2EID, &p.TXID, &p.PayerName, &p.PayerDocument, &p.PayerBank, &p.CreatedAt, &p.UpdatedAt)
	if err != nil { return nil, err }
	p.Method = domain.PaymentMethod(method)
	p.Status = domain.PaymentStatus(status)
	return &p, nil
}

type OutboxEvent struct {
	EventType string `json:"eventType"`
	PaymentID string `json:"paymentId"`
}

func (r Repo) CreatePaymentWithOutbox(ctx context.Context, p domain.PaymentIntent) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
    INSERT INTO payment_intent(payment_id, order_id, idempotency_key, amount_cents, currency, method, status, provider,
                              payer_name, payer_document, payer_bank)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
  `, p.PaymentID, p.OrderID, p.IdempotencyKey, p.AmountCents, p.Currency, string(p.Method), string(p.Status), p.Provider,
		p.PayerName, p.PayerDocument, p.PayerBank)
	if err != nil { return err }

	payload, _ := json.Marshal(OutboxEvent{EventType: "PaymentRequested", PaymentID: p.PaymentID})
	_, err = tx.Exec(ctx, `INSERT INTO outbox(event_type, payload_json) VALUES ($1,$2)`, "PaymentRequested", string(payload))
	if err != nil { return err }

	return tx.Commit(ctx)
}

func (r Repo) UpdateStatus(ctx context.Context, paymentId string, status domain.PaymentStatus, provider string, e2e, txid *string) error {
	_, err := r.DB.Exec(ctx, `
    UPDATE payment_intent
       SET status=$2, provider=$3, e2e_id=COALESCE($4,e2e_id), txid=COALESCE($5,txid), updated_at=now()
     WHERE payment_id=$1`, paymentId, string(status), provider, e2e, txid)
	return err
}

func (r Repo) FetchUnpublishedOutbox(ctx context.Context, limit int) ([]struct{ID int64; Payload string}, error) {
	rows, err := r.DB.Query(ctx, `SELECT id, payload_json FROM outbox WHERE published_at IS NULL ORDER BY id LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var res []struct{ID int64; Payload string}
	for rows.Next() {
		var id int64
		var payload string
		if err := rows.Scan(&id, &payload); err != nil { return nil, err }
		res = append(res, struct{ID int64; Payload string}{ID:id, Payload:payload})
	}
	return res, nil
}

func (r Repo) MarkOutboxPublished(ctx context.Context, id int64) error {
	_, err := r.DB.Exec(ctx, `UPDATE outbox SET published_at=now() WHERE id=$1`, id)
	return err
}


type ProviderFields struct {
	Provider string
	E2EID *string
	TXID *string
	CardBrand *string
	CardMaskedPAN *string
	CardAuthCode *string
}

func (r Repo) UpdateProviderFields(ctx context.Context, paymentId string, status domain.PaymentStatus, f ProviderFields) error {
	_, err := r.DB.Exec(ctx, `
    UPDATE payment_intent
       SET status=$2, provider=$3,
           e2e_id=COALESCE($4,e2e_id),
           txid=COALESCE($5,txid),
           card_brand=COALESCE($6,card_brand),
           card_masked_pan=COALESCE($7,card_masked_pan),
           card_auth_code=COALESCE($8,card_auth_code),
           updated_at=now()
     WHERE payment_id=$1`,
		paymentId, string(status), f.Provider, f.E2EID, f.TXID, f.CardBrand, f.CardMaskedPAN, f.CardAuthCode)
	return err
}
