package providers

import (
	"context"
	"errors"
	"strings"

	"github.com/you/clickpay-go-poc/internal/pix"
)

type PixMock struct{}

func (p PixMock) Name() string { return "pixmock" }
func (p PixMock) Supports(m Method) bool { return m == MethodPIX }

func (p PixMock) Authorize(ctx context.Context, method Method, req AuthorizeRequest) (AuthorizeResponse, error) {
	if method != MethodPIX { return AuthorizeResponse{}, errors.New("unsupported") }
	if strings.TrimSpace(req.PixKey) == "" { return AuthorizeResponse{}, errors.New("missing pix.key") }

	resp := pix.CreateCharge(pix.CreatePixRequest{
		PaymentID: req.PaymentID,
		AmountCents: req.AmountCents,
		Currency: req.Currency,
		Key: req.PixKey,
		PayerMessage: req.PixPayerMessage,
		ExpiresInSeconds: req.PixExpiresSec,
	})
	tx := resp.TxID
	// PIX geralmente fica PENDING até webhook
	return AuthorizeResponse{Provider: "pixmock", Status: "PENDING", TxID: &tx}, nil
}
