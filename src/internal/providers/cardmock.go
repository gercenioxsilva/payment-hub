package providers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type CardMock struct{}

func (c CardMock) Name() string { return "cardmock" }
func (c CardMock) Supports(m Method) bool { return m == MethodCardCredit || m == MethodCardDebit }

func (c CardMock) Authorize(ctx context.Context, method Method, req AuthorizeRequest) (AuthorizeResponse, error) {
	if !c.Supports(method) { return AuthorizeResponse{}, errors.New("unsupported") }
	if len(strings.TrimSpace(req.CardNumber)) < 12 { return AuthorizeResponse{}, errors.New("invalid cardNumber") }
	if req.CardExpMonth < 1 || req.CardExpMonth > 12 { return AuthorizeResponse{}, errors.New("invalid expMonth") }
	if req.CardExpYear < time.Now().Year() { return AuthorizeResponse{}, errors.New("card expired") }
	if strings.TrimSpace(req.CardCVV) == "" { return AuthorizeResponse{}, errors.New("missing cvv") }

	// simple deterministic-ish auth
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	auth := fmt.Sprintf("A%06d", r.Intn(1000000))
	masked := "**** **** **** " + req.CardNumber[len(req.CardNumber)-4:]
	brand := req.CardBrand
	status := "AUTHORIZED"
	return AuthorizeResponse{
		Provider: "cardmock",
		Status: status,
		AuthCode: &auth,
		MaskedPAN: &masked,
		Brand: &brand,
	}, nil
}
