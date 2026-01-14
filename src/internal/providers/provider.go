package providers

import "context"

type Method string

const (
	MethodPIX        Method = "PIX"
	MethodCardCredit Method = "CARD_CREDIT"
	MethodCardDebit  Method = "CARD_DEBIT"
)

type AuthorizeRequest struct {
	PaymentID   string
	AmountCents int64
	Currency    string

	// PIX
	PixKey          string
	PixPayerMessage string
	PixExpiresSec   int

	// CARD
	CardHolderName string
	CardNumber     string
	CardExpMonth   int
	CardExpYear    int
	CardCVV        string
	CardBrand      string
}

type AuthorizeResponse struct {
	Provider string
	Status   string // PENDING | AUTHORIZED | PAID | FAILED
	TxID     *string
	E2EID    *string
	AuthCode *string
	MaskedPAN *string
	Brand    *string
}

type Provider interface {
	Name() string
	Supports(method Method) bool
	Authorize(ctx context.Context, method Method, req AuthorizeRequest) (AuthorizeResponse, error)
}
