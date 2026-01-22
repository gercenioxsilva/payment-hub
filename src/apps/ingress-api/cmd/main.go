package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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


// StartPayment types (generic initializer)
type StartPaymentRequest struct {
    TokenSession     string                 `json:"tokenSession"`
    PartnerUniqueID  string                 `json:"partnerUniqueID"`
    Customer         StartCustomer          `json:"customer"`
    Vendor           StartVendor            `json:"vendor"`
    PaymentItems     StartPaymentItems      `json:"paymentItems"`
    PaymentData      StartPaymentData       `json:"paymentData"`
    FrontData        StartFrontData         `json:"frontData"`
    IsMerchantInitiated bool                `json:"isMerchantInitiated"`
    ExtraInfo        string                 `json:"extraInfo"`
}

type StartCustomer struct {
    ID                        string `json:"iD"`
    Email                     string `json:"email"`
    Name                      string `json:"name"`
    Document                  string `json:"document"`
    DocumentType              string `json:"documentType"`
    BirthDate                 string `json:"birthDate"`
    Gender                    string `json:"gender"`
    DaysSinceRegistration     int    `json:"daysSinceRegistration"`
    DaysSinceLastLogin        int    `json:"daysSinceLastLogin"`
    DaysSinceLastPasswordChange int  `json:"daysSinceLastPasswordChange"`
    DaysSinceLastPurchase     int    `json:"daysSinceLastPurchase"`
    DaysSinceFirstPurchase    int    `json:"daysSinceFirstPurchase"`
}

type StartVendor struct {
    MCC                     string `json:"mCC"`
    Name                    string `json:"name"`
    Code                    string `json:"code"`
    Industry                string `json:"industry"`
    Document                string `json:"document"`
    DocumentType            string `json:"documentType"`
    DaysSinceRegistration   int    `json:"daysSinceRegistration"`
    DaysSinceLastSell       int    `json:"daysSinceLastSell"`
    DaysSinceFirstSell      int    `json:"daysSinceFirstSell"`
}

type StartPaymentItems struct {
    Items []StartItem `json:"items"`
}

type StartItem struct {
    DetailUniqueID   string   `json:"detailUniqueID"`
    SKU              string   `json:"sku"`
    EAN              string   `json:"ean"`
    Amount           float64  `json:"amount"`
    ProductID        int      `json:"productID"`
    ProductDescription string `json:"productDescription"`
    CategoryID       int      `json:"categoryID"`
    CategoryName     string   `json:"categoryName"`
    ItemQuantity     int      `json:"itemQuantity"`
    Split            *StartSplit `json:"split,omitempty"`
    AntiFraud        *StartAntiFraud `json:"antiFraud,omitempty"`
    PresentMessage   string   `json:"presentMessage"`
}

type StartSplit struct {
    MerchantID           string  `json:"merchantID"`
    MerchantDocument     string  `json:"merchantDocument"`
    MerchantDocumentType string  `json:"merchantDocumentType"`
    Percent              float64 `json:"percent"`
    Amount               float64 `json:"amount"`
}

type StartAntiFraud struct {
    EAN                         string `json:"ean"`
    DeliveryAddressee           string `json:"deliveryAddressee"`
    IsEmailConfirmed            bool   `json:"isEmailConfirmed"`
    WasCardChanged              bool   `json:"wasCardChanged"`
    IsPhoneNumberConfirmed      bool   `json:"isPhoneNumberConfirmed"`
    LoginCredential             string `json:"loginCredential"`
    IsVIPClient                 bool   `json:"isVIPClient"`
    IsEmployeeClient            bool   `json:"isEmployeeClient"`
    WasAccountModified          bool   `json:"wasAccountModified"`
    DaysSinceLastAccountChange  int    `json:"daysSinceLastAccountChange"`
    FidelityNumber              string `json:"fidelityNumber"`
    DaysSinceFidelityRegistration int  `json:"daysSinceFidelityRegistration"`
    DaysSinceLastPointsExchange int    `json:"daysSinceLastPointsExchange"`
    FidelityBalance             int    `json:"fidelityBalance"`
    AmountPointsLastExchange    int    `json:"amountPointsLastExchange"`
    PhoneChargesInLast30Days    int    `json:"phoneChargesInLast30Days"`
    MinutesChargedInLast30Days  int    `json:"minutesChargedInLast30Days"`
    Data                        *StartKeyValue `json:"data,omitempty"`
}

type StartKeyValue struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type StartPaymentData struct {
    PaymentMethods []StartPaymentMethod `json:"paymentMethods"`
    AntiFraud      *StartAntiFraud     `json:"antiFraud,omitempty"`
    DeliveryAddress *StartAddress      `json:"deliveryAddress,omitempty"`
    CountryCode    string              `json:"countrycode"`
    Amount         float64             `json:"amount"`
    SalesChannel   string              `json:"salesChannel"`
}

type StartPaymentMethod struct {
    PaymentMethodType string           `json:"paymentMethodType"`
    Amount            float64          `json:"amount"`
    Installments      int              `json:"installments"`
    CardInfo          *StartCardInfo   `json:"cardInfo,omitempty"`
    BoletoInfo        *StartBoletoInfo `json:"boletoInfo,omitempty"`
    Customer          *StartCustomer   `json:"customer,omitempty"`
    Pix               *StartPixInfo    `json:"pix,omitempty"`
    SoftDescriptor    string           `json:"softDescriptor"`
}

type StartCardInfo struct {
    Token            string            `json:"token"`
    TokenProvider    string            `json:"tokenProvider"`
    CardHolderName   string            `json:"cardHolderName"`
    Alias            string            `json:"alias"`
    ExpirationMonth  int               `json:"expirationMonth"`
    ExpirationYear   int               `json:"expirationYear"`
    BillingInfo      *StartBillingInfo `json:"billingInfo,omitempty"`
    ExternalProviderInfo string        `json:"externalProviderInfo"`
    BrandName        string            `json:"brandName"`
    TokenSingleUse   int               `json:"tokenSingleUse"`
    SaveCard         bool              `json:"saveCard"`
}

type StartBillingInfo struct {
    Number       string        `json:"number"`
    Document     string        `json:"document"`
    DocumentType string        `json:"documentType"`
    Name         string        `json:"name"`
    Phone        string        `json:"phone"`
    Address      *StartAddress `json:"address,omitempty"`
}

type StartBoletoInfo struct {
    BillingInfo *StartBillingInfo `json:"billingInfo,omitempty"`
}

type StartPixInfo struct {
    Name           string `json:"name"`
    Document       string `json:"document"`
    DocumentType   string `json:"documentType"`
    ExpirationSeconds int  `json:"expirationSeconds"`
}

type StartAddress struct {
    Street       string `json:"street"`
    Number       string `json:"number"`
    Complement   string `json:"complement"`
    Neighborhood string `json:"neighborhood"`
    City         string `json:"city"`
    State        string `json:"state"`
    Country      string `json:"country"`
    PostalCode   string `json:"postalCode"`
    Phone        string `json:"phone"`
}

type StartFrontData struct {
    SessionID      string `json:"sessionID"`
    IPAddress      string `json:"ipAddress"`
    AcceptHeader   string `json:"acceptHeader"`
    UserAgent      string `json:"userAgent"`
    CookiesAccepted bool  `json:"cookiesAccepted"`
}

// StartPaymentResponse (novo formato)
type APIMessage struct {
    Source  int32  `json:"source"`
    Code    string `json:"code"`
    Message string `json:"message"`
    Info    string `json:"info"`
}

type RedirectInfo struct {
    URL string `json:"url"`
}

type AcquirerInfo struct {
    Name    string `json:"name"`
    Status  string `json:"status"`
    Message string `json:"message"`
}

type AntifraudInfo struct {
    Name    string `json:"name"`
    Status  string `json:"status"`
    Message string `json:"message"`
}

type PixReturnInfo struct {
    QRContent    string `json:"qRContent"`
    QRCopyPaste  string `json:"qrCopyPaste"`
    QRImage      string `json:"qRImage"`
}

type CryptoReturnInfo struct {
    CoinValue        string `json:"coinValue"`
    CoinRateCurrency string `json:"coinRateCurrency"`
    CoinAddr         string `json:"coinAddr"`
    CoinQRCodeURL    string `json:"coinQRCodeUrl"`
    Coin             string `json:"coin"`
}

type MethodReturn struct {
    MethodType  string             `json:"methodType"`
    Status      string             `json:"status"`
    MethodID    int16              `json:"methodId"`
    OperationID string             `json:"operationId"`
    MethodKey   string             `json:"methodKey"`
    Message     APIMessage         `json:"message"`
    Redirect    RedirectInfo       `json:"redirectInfo"`
    Acquirer    []AcquirerInfo     `json:"acquirer"`
    Antifraud   []AntifraudInfo    `json:"antifraud"`
    PixInfo     *PixReturnInfo     `json:"pixInfo,omitempty"`
    CryptoInfo  *CryptoReturnInfo  `json:"cryptoInfo,omitempty"`
}

type StartPaymentResponse struct {
    Status          string          `json:"status"`
    Methods         []MethodReturn  `json:"methods"`
    PaymentKey      string          `json:"paymentKey"`
    PartnerUniqueId string          `json:"partnerUniqueId"`
    Code            int32           `json:"code"`
    Message         APIMessage      `json:"message"`
    OperationID     string          `json:"operationId"`
}


func main() {
    cfg := config.Load()
    ctx := context.Background()

    db, err := persistence.Connect(ctx, cfg.DBURL)
    if err != nil { log.Fatal(err) }
    repo := persistence.Repo{DB: db}

    v := validator.New()
    r := chi.NewRouter()

    // Liveness/Readiness endpoint
    r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
        if err := db.Ping(req.Context()); err != nil {
            http.Error(w, "unhealthy: "+err.Error(), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type","application/json")
        json.NewEncoder(w).Encode(map[string]any{"status":"ok"})
    })

	// Generic payment initializer for the extended schema
	r.Post("/payments/init", func(w http.ResponseWriter, req *http.Request) {
		idem := req.Header.Get("Idempotency-Key")
		if idem == "" { http.Error(w, "missing Idempotency-Key", 400); return }

		var body StartPaymentRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil { http.Error(w, err.Error(), 400); return }

		orderID := strings.TrimSpace(body.PartnerUniqueID)
		if orderID == "" { http.Error(w, "partnerUniqueID is required", 400); return }

        if existing, err := repo.FindByOrderAndIdem(req.Context(), orderID, idem); err == nil && existing != nil {
            // Replay must return the V2 envelope as well
            mType := strings.ToUpper(string(existing.Method))
            mr := MethodReturn{
                MethodType: mType,
                Status:     "Q",
                MethodID:   1,
                OperationID: existing.PaymentID,
                MethodKey:   existing.PaymentID,
                Message: APIMessage{Source: 0, Code: "OK", Message: "REPLAY", Info: ""},
                Redirect: RedirectInfo{URL: ""},
                Acquirer: []AcquirerInfo{},
                Antifraud: []AntifraudInfo{},
            }
            if strings.EqualFold(mType, "PIX") {
                mr.PixInfo = &PixReturnInfo{}
            }
            resp := StartPaymentResponse{
                Status:          "Q",
                Methods:         []MethodReturn{mr},
                PaymentKey:      existing.PaymentID,
                PartnerUniqueId: orderID,
                Code:            200,
                Message:         APIMessage{Source: 0, Code: "OK", Message: "QUEUED", Info: ""},
                OperationID:     existing.PaymentID,
            }
            w.Header().Set("Content-Type","application/json")
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(resp)
            return
        }

		method := "INIT"
		provider := "orchestrator"
		if len(body.PaymentData.PaymentMethods) > 0 {
			pm := body.PaymentData.PaymentMethods[0]
			if pm.PaymentMethodType != "" { method = strings.ToUpper(pm.PaymentMethodType) }
		}

		amountCents := int64(body.PaymentData.Amount * 100)
		currency := "BRL"

		paymentID := ulid.Make().String()
		p := domain.PaymentIntent{
			PaymentID: paymentID,
			OrderID: orderID,
			IdempotencyKey: idem,
			AmountCents: amountCents,
			Currency: currency,
			Method: domain.PaymentMethod(method),
			Status: domain.StatusQueued,
			Provider: provider,
		}

		// Optional payer details from top-level customer
		if body.Customer.Name != "" { p.PayerName = &body.Customer.Name }
		if body.Customer.Document != "" { p.PayerDocument = &body.Customer.Document }

		if err := repo.CreatePaymentWithOutbox(req.Context(), p); err != nil { http.Error(w, fmt.Sprintf("persist: %v", err), 500); return }

		// montar resposta no novo formato
		mr := MethodReturn{
			MethodType: strings.ToUpper(method),
			Status:     "Q",
			MethodID:   1,
			OperationID: paymentID,
			MethodKey:   paymentID,
			Message: APIMessage{Source: 0, Code: "QUEUED", Message: "", Info: ""},
			Redirect: RedirectInfo{URL: ""},
			Acquirer: []AcquirerInfo{},
			Antifraud: []AntifraudInfo{},
		}
		if len(body.PaymentData.PaymentMethods) > 0 {
			pm := body.PaymentData.PaymentMethods[0]
			if strings.EqualFold(pm.PaymentMethodType, "PIX") {
				mr.PixInfo = &PixReturnInfo{}
			}
		}

        resp := StartPaymentResponse{
            Status:          "Q",
            Methods:         []MethodReturn{mr},
            PaymentKey:      paymentID,
            PartnerUniqueId: orderID,
            Code:            200,
            Message:         APIMessage{Source: 0, Code: "OK", Message: "QUEUED", Info: ""},
            OperationID:     paymentID,
        }

        w.Header().Set("Content-Type","application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(resp)
    })

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
