package pix

import (
	"fmt"
	"time"
)

type CreatePixRequest struct {
	PaymentID        string
	AmountCents      int64
	Currency         string
	Key              string
	PayerMessage     string
	ExpiresInSeconds int
}

type CreatePixResponse struct {
	Provider     string    `json:"provider"`
	TxID         string    `json:"txid"`
	ExpiresAt    time.Time `json:"expiresAt"`
	QrCodeBase64 string    `json:"qrCodeBase64"`
	BrCode       string    `json:"brCode"`
	Location     string    `json:"location"`
}

func CreateCharge(req CreatePixRequest) CreatePixResponse {
	txid := fmt.Sprintf("TX-%s", req.PaymentID[:8])
	expires := time.Now().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
	brCode := fmt.Sprintf("00020101021226820014br.gov.bcb.pix01%02d%s5204000053039865405%0.2f5802BR5920CLICKPAY POC6009SAO PAULO62170513%s6304ABCD",
		len(req.Key), req.Key, float64(req.AmountCents)/100.0, txid)
	qr := "UE9DX1FSX0NPREVfQkFTRTY0"
	return CreatePixResponse{Provider:"pixmock", TxID:txid, ExpiresAt:expires, QrCodeBase64:qr, BrCode:brCode, Location:"https://psp.mock/pix/"+txid}
}
