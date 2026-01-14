package domain

import "time"

type PaymentStatus string
const (
	StatusCreated    PaymentStatus = "CREATED"
	StatusQueued     PaymentStatus = "QUEUED"
	StatusProcessing PaymentStatus = "PROCESSING"
	StatusPending    PaymentStatus = "PENDING"
	StatusPaid       PaymentStatus = "PAID"
	StatusFailed     PaymentStatus = "FAILED"
)

type PaymentMethod string
const (
	MethodPIX PaymentMethod = "PIX"
)

type PaymentIntent struct {
	PaymentID      string
	OrderID        string
	IdempotencyKey string
	AmountCents    int64
	Currency       string
	Method         PaymentMethod
	Status         PaymentStatus
	Provider       string
	E2EID          *string
	TXID           *string
	PayerName      *string
	PayerDocument  *string
	PayerBank      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
