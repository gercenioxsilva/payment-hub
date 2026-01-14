package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/oklog/ulid/v2"

	"github.com/you/clickpay-go-poc/internal/config"
	"github.com/you/clickpay-go-poc/internal/domain"
	"github.com/you/clickpay-go-poc/internal/persistence"
	"github.com/you/clickpay-go-poc/internal/pix"
)

type Money struct {
	ValueCents int64  `json:"valueCents" validate:"gt=0"`
	Currency   string `json:"currency" validate:"required,oneof=BRL"`
}

type Payer struct {
	Name     string `json:"name" validate:"required"`
	Document string `json:"document" validate:"required"`
	Bank     string `json:"bank" validate:"required"`
}

type PixData struct {
	Key              string `json:"key" validate:"required"`
	PayerMessage     string `json:"payerMessage" validate:"required"`
	ExpiresInSeconds int    `json:"expiresInSeconds" validate:"gte=60,lte=86400"`
}

type CreatePixPaymentRequest struct {
	OrderID string  `json:"orderId" validate:"required"`
	Amount  Money   `json:"amount" validate:"required"`
	Payer   Payer   `json:"payer" validate:"required"`
	Pix     PixData `json:"pix" validate:"required"`
}

type CreatePixPaymentResponse struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
	Method    string `json:"method"`
	Provider  string `json:"provider"`

	Pix struct {
		TxID         string    `json:"txid"`
		ExpiresAt    time.Time `json:"expiresAt"`
		QrCodeBase64 string    `json:"qrCodeBase64"`
		BrCode       string    `json:"brCode"`
		Location     string    `json:"location"`
	} `json:"pix"`
}


type CardData struct {
	HolderName string `json:"holderName" validate:"required"`
	Number     string `json:"number" validate:"required"`
	ExpMonth   int    `json:"expMonth" validate:"gte=1,lte=12"`
	ExpYear    int    `json:"expYear" validate:"gte=2024"`
	CVV        string `json:"cvv" validate:"required"`
	Brand      string `json:"brand" validate:"required"`
}

type CreateCardPaymentRequest struct {
	OrderID string `json:"orderId" validate:"required"`
	Amount  Money  `json:"amount" validate:"required"`
	Card    CardData `json:"card" validate:"required"`
}

type CreateCardPaymentResponse struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
	Method    string `json:"method"`
	Provider  string `json:"provider"`
	Card *struct {
		MaskedPan string `json:"maskedPan"`
		Brand string `json:"brand"`
		AuthCode string `json:"authCode"`
	} `json:"card,omitempty"`
}


func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := persistence.Connect(ctx, cfg.DBURL)
	if err != nil { log.Fatal(err) }
	repo := persistence.Repo{DB: db}

	v := validator.New()
	r := chi.NewRouter()

	r.Post("/pix/payments", func(w http.ResponseWriter, req *http.Request) {
		idem := req.Header.Get("Idempotency-Key")
		if idem == "" { http.Error(w, "missing Idempotency-Key", 400); return }

		var body CreatePixPaymentRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil { http.Error(w, err.Error(), 400); return }
		if err := v.Struct(body); err != nil { http.Error(w, err.Error(), 400); return }

		if existing, err := repo.FindByOrderAndIdem(req.Context(), body.OrderID, idem); err == nil && existing != nil {
			out := map[string]any{"paymentId": existing.PaymentID, "status": existing.Status, "replay": true}
			w.Header().Set("Content-Type","application/json")
			json.NewEncoder(w).Encode(out)
			return
		}

		paymentID := ulid.Make().String()
		provider := "pixmock"

		p := domain.PaymentIntent{
			PaymentID: paymentID,
			OrderID: body.OrderID,
			IdempotencyKey: idem,
			AmountCents: body.Amount.ValueCents,
			Currency: body.Amount.Currency,
			Method: domain.MethodPIX,
			Status: domain.StatusQueued,
			Provider: provider,
			PayerName: &body.Payer.Name,
			PayerDocument: &body.Payer.Document,
			PayerBank: &body.Payer.Bank,
		}

		if err := repo.CreatePaymentWithOutbox(req.Context(), p); err != nil { http.Error(w, err.Error(), 500); return }

		pixResp := pix.CreateCharge(pix.CreatePixRequest{
			PaymentID: paymentID,
			AmountCents: body.Amount.ValueCents,
			Currency: body.Amount.Currency,
			Key: body.Pix.Key,
			PayerMessage: body.Pix.PayerMessage,
			ExpiresInSeconds: body.Pix.ExpiresInSeconds,
		})

		resp := CreatePixPaymentResponse{PaymentID: paymentID, Status: "QUEUED", Method: "PIX", Provider: provider}
		resp.Pix.TxID = pixResp.TxID
		resp.Pix.ExpiresAt = pixResp.ExpiresAt
		resp.Pix.QrCodeBase64 = pixResp.QrCodeBase64
		resp.Pix.BrCode = pixResp.BrCode
		resp.Pix.Location = pixResp.Location

		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(resp)
	})


	r.Post("/card/credit/payments", func(w http.ResponseWriter, req *http.Request) {
		createCard(w, req, repo, v, "CARD_CREDIT")
	})
	r.Post("/card/debit/payments", func(w http.ResponseWriter, req *http.Request) {
		createCard(w, req, repo, v, "CARD_DEBIT")
	})

	r.Get("/payments/{paymentId}", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "paymentId")
		p, err := repo.FindByPaymentID(req.Context(), id)
		if err != nil { http.Error(w, err.Error(), 404); return }
		w.Header().Set("Content-Type","application/json")
		json.NewEncoder(w).Encode(p)
	})

	addr := ":" + cfg.IngressPort
	log.Println("ingress-api listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}



func createCard(w http.ResponseWriter, req *http.Request, repo persistence.Repo, v *validator.Validate, method string) {
	idem := req.Header.Get("Idempotency-Key")
	if idem == "" { http.Error(w, "missing Idempotency-Key", 400); return }

	var body CreateCardPaymentRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil { http.Error(w, err.Error(), 400); return }
	if err := v.Struct(body); err != nil { http.Error(w, err.Error(), 400); return }

	if existing, err := repo.FindByOrderAndIdem(req.Context(), body.OrderID, idem); err == nil && existing != nil {
		out := map[string]any{"paymentId": existing.PaymentID, "status": existing.Status, "replay": true}
		w.Header().Set("Content-Type","application/json")
		json.NewEncoder(w).Encode(out)
		return
	}

	paymentID := ulid.Make().String()
	provider := "cardmock"

	m := domain.PaymentMethod(method)
	p := domain.PaymentIntent{
		PaymentID: paymentID,
		OrderID: body.OrderID,
		IdempotencyKey: idem,
		AmountCents: body.Amount.ValueCents,
		Currency: body.Amount.Currency,
		Method: m,
		Status: domain.StatusQueued,
		Provider: provider,
	}

	if err := repo.CreatePaymentWithOutbox(req.Context(), p); err != nil { http.Error(w, err.Error(), 500); return }

	resp := CreateCardPaymentResponse{PaymentID: paymentID, Status: "QUEUED", Method: method, Provider: provider}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

