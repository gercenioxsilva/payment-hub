package providers

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCardMockAuthorizeValidatesInput(t *testing.T) {
	ctx := context.Background()
	mock := CardMock{}

	_, err := mock.Authorize(ctx, MethodPIX, AuthorizeRequest{})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported error, got %v", err)
	}

	_, err = mock.Authorize(ctx, MethodCardCredit, AuthorizeRequest{CardNumber: "123"})
	if err == nil || !strings.Contains(err.Error(), "invalid cardNumber") {
		t.Fatalf("expected invalid cardNumber error, got %v", err)
	}

	_, err = mock.Authorize(ctx, MethodCardCredit, AuthorizeRequest{CardNumber: "4111111111111", CardExpMonth: 13})
	if err == nil || !strings.Contains(err.Error(), "invalid expMonth") {
		t.Fatalf("expected invalid expMonth error, got %v", err)
	}

	_, err = mock.Authorize(ctx, MethodCardCredit, AuthorizeRequest{
		CardNumber: "4111111111111",
		CardExpMonth: 12,
		CardExpYear: time.Now().Year() - 1,
	})
	if err == nil || !strings.Contains(err.Error(), "card expired") {
		t.Fatalf("expected card expired error, got %v", err)
	}

	_, err = mock.Authorize(ctx, MethodCardCredit, AuthorizeRequest{
		CardNumber: "4111111111111",
		CardExpMonth: 12,
		CardExpYear: time.Now().Year() + 1,
	})
	if err == nil || !strings.Contains(err.Error(), "missing cvv") {
		t.Fatalf("expected missing cvv error, got %v", err)
	}
}

func TestCardMockAuthorizeSuccess(t *testing.T) {
	ctx := context.Background()
	mock := CardMock{}

	resp, err := mock.Authorize(ctx, MethodCardCredit, AuthorizeRequest{
		CardNumber: "4111111111111111",
		CardExpMonth: 12,
		CardExpYear: time.Now().Year() + 1,
		CardCVV: "123",
		CardBrand: "VISA",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "cardmock" || resp.Status != "AUTHORIZED" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.AuthCode == nil || !strings.HasPrefix(*resp.AuthCode, "A") {
		t.Fatalf("expected auth code, got %+v", resp.AuthCode)
	}
	if resp.MaskedPAN == nil || !strings.HasSuffix(*resp.MaskedPAN, "1111") {
		t.Fatalf("expected masked pan ending in 1111, got %+v", resp.MaskedPAN)
	}
	if resp.Brand == nil || *resp.Brand != "VISA" {
		t.Fatalf("expected brand VISA, got %+v", resp.Brand)
	}
}
