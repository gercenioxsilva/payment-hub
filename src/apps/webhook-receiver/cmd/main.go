package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/you/clickpay-go-poc/internal/config"
	"github.com/you/clickpay-go-poc/internal/domain"
	"github.com/you/clickpay-go-poc/internal/persistence"
)


type CardWebhook struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"` // PAID | FAILED
}


type PixWebhook struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
	E2EID     string `json:"e2eId"`
	TXID      string `json:"txid"`
}

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := persistence.Connect(ctx, cfg.DBURL)
	if err != nil { log.Fatal(err) }
	repo := persistence.Repo{DB: db}

	r := chi.NewRouter()

	r.Post("/webhooks/pix", func(w http.ResponseWriter, req *http.Request) {
		var wh PixWebhook
		if err := json.NewDecoder(req.Body).Decode(&wh); err != nil { http.Error(w, err.Error(), 400); return }
		if wh.PaymentID == "" || wh.Status == "" { http.Error(w, "missing fields", 400); return }

		provider := "pixmock"
		var st domain.PaymentStatus
		switch wh.Status {
		case "PAID":
			st = domain.StatusPaid
		case "FAILED":
			st = domain.StatusFailed
		default:
			http.Error(w, "invalid status", 400); return
		}
		e2e := wh.E2EID
		txid := wh.TXID
		if err := repo.UpdateStatus(req.Context(), wh.PaymentID, st, provider, &e2e, &txid); err != nil {
			http.Error(w, err.Error(), 500); return
		}
		w.WriteHeader(200)
	})


	r.Post("/webhooks/card", func(w http.ResponseWriter, req *http.Request) {
		var wh CardWebhook
		if err := json.NewDecoder(req.Body).Decode(&wh); err != nil { http.Error(w, err.Error(), 400); return }
		if wh.PaymentID == "" || wh.Status == "" { http.Error(w, "missing fields", 400); return }
		provider := "cardmock"
		var st domain.PaymentStatus
		switch wh.Status {
		case "PAID":
			st = domain.StatusPaid
		case "FAILED":
			st = domain.StatusFailed
		default:
			http.Error(w, "invalid status", 400); return
		}
		if err := repo.UpdateStatus(req.Context(), wh.PaymentID, st, provider, nil, nil); err != nil {
			http.Error(w, err.Error(), 500); return
		}
		w.WriteHeader(200)
	})


	addr := ":" + cfg.WebhookPort
	log.Println("webhook-receiver listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
