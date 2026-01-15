package providers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/you/clickpay-go-poc/internal/pix"
)

func TestPixMockAuthorizeValidatesInput(t *testing.T) {
	ctx := context.Background()
	mock := PixMock{}

	_, err := mock.Authorize(ctx, MethodCardCredit, AuthorizeRequest{})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported error, got %v", err)
	}

	_, err = mock.Authorize(ctx, MethodPIX, AuthorizeRequest{})
	if err == nil || !strings.Contains(err.Error(), "missing pix.key") {
		t.Fatalf("expected missing pix.key error, got %v", err)
	}
}

func TestPixMockAuthorizeSuccess(t *testing.T) {
	ctx := context.Background()
	mock := PixMock{}

	resp, err := mock.Authorize(ctx, MethodPIX, AuthorizeRequest{
		PaymentID: "payment-12345678",
		AmountCents: 2500,
		Currency: "BRL",
		PixKey: "email@merchant.com",
		PixPayerMessage: "Pedido 1",
		PixExpiresSec: 600,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "pixmock" || resp.Status != "PENDING" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.TxID == nil || !strings.HasPrefix(*resp.TxID, "TX-") {
		t.Fatalf("expected txid with TX- prefix, got %+v", resp.TxID)
	}
}

func TestCreateChargeBuildsDeterministicFields(t *testing.T) {
	start := time.Now()
	resp := pix.CreateCharge(pix.CreatePixRequest{
		PaymentID: "payment-abcdef12",
		AmountCents: 990,
		Currency: "BRL",
		Key: "email@merchant.com",
		PayerMessage: "Pedido 2",
		ExpiresInSeconds: 120,
	})
	end := time.Now()

	if !strings.HasPrefix(resp.TxID, "TX-") {
		t.Fatalf("expected txid prefix, got %s", resp.TxID)
	}
	if resp.Location == "" || !strings.Contains(resp.Location, resp.TxID) {
		t.Fatalf("expected location with txid, got %s", resp.Location)
	}
	if resp.ExpiresAt.Before(start) || resp.ExpiresAt.After(end.Add(2*time.Minute+time.Second)) {
		t.Fatalf("unexpected expiresAt: %s", resp.ExpiresAt)
	}
	if resp.BrCode == "" || !strings.Contains(resp.BrCode, resp.TxID) {
		t.Fatalf("expected brcode to include txid, got %s", resp.BrCode)
	}
}
