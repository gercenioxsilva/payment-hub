package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/cenkalti/backoff/v4"
	"github.com/sony/gobreaker"

	"github.com/you/clickpay-go-poc/internal/config"
	"github.com/you/clickpay-go-poc/internal/domain"
	"github.com/you/clickpay-go-poc/internal/persistence"
	"github.com/you/clickpay-go-poc/internal/providers"
	"github.com/you/clickpay-go-poc/internal/queue"
)

type OutboxEvent struct {
	EventType string `json:"eventType"`
	PaymentID string `json:"paymentId"`
}

type sem chan struct{}
func newSem(n int) sem { s := make(chan struct{}, n); return s }
func (s sem) acquire() { s <- struct{}{} }
func (s sem) release() { <-s }

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := persistence.Connect(ctx, cfg.DBURL)
	if err != nil { log.Fatal(err) }
	repo := persistence.Repo{DB: db}

	policy, err := config.LoadPolicy("config/policy.yml")
	if err != nil { log.Fatal("policy:", err) }

	sqsClient, err := queue.New(ctx, cfg.SQSEndpoint, cfg.AWSRegion, cfg.AccessKey, cfg.SecretKey)
	if err != nil { log.Fatal(err) }

	mainURL := mustQueueURL(ctx, sqsClient.SQS, cfg.SQSQueue)
	dlqName := cfg.SQSQueue + "-dlq"
	dlqURL := mustQueueURL(ctx, sqsClient.SQS, dlqName)

	log.Println("orchestrator-worker started, main:", mainURL, "dlq:", dlqURL)

	// Providers registry
	provList := []providers.Provider{
		providers.PixMock{},
		providers.CardMock{},
	}

	// Bulkheads (max in-flight per provider)
	bulk := map[string]sem{
		"pixmock":  newSem(envInt("BULKHEAD_PIX", 50)),
		"cardmock": newSem(envInt("BULKHEAD_CARD", 50)),
	}

	// Circuit breakers per provider
	cb := map[string]*gobreaker.CircuitBreaker{}
	for _, p := range provList {
		name := p.Name()
		cb[name] = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name: name,
			MaxRequests: 5,
			Interval: 30 * time.Second,
			Timeout: 15 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				// abre circuito com taxa de falha > 50% e pelo menos 10 reqs
				if counts.Requests < 10 { return false }
				failRate := float64(counts.TotalFailures) / float64(counts.Requests)
				return failRate >= 0.5
			},
		})
	}

	maxAttempts := policy.Defaults.Retries.MaxAttempts
	if maxAttempts < 1 { maxAttempts = 1 }
	maxReceive := envInt("MAX_RECEIVE_COUNT", 3)

	for {
		msgs, err := sqsClient.SQS.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl: &mainURL,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds: 10,
			VisibilityTimeout: 30,
			AttributeNames: []sqs.QueueAttributeName{"All"},
		})
		if err != nil { log.Println("receive:", err); continue }
		if len(msgs.Messages) == 0 { continue }

		for _, m := range msgs.Messages {
			// handle each message sequentially in POC; scale via multiple replicas in k8s
			err := handleMessage(ctx, repo, policy, provList, bulk, cb, m.Body, maxAttempts)
			if err == nil {
				_, _ = sqsClient.SQS.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: &mainURL, ReceiptHandle: m.ReceiptHandle})
				continue
			}

			rc := approxReceiveCount(m.Attributes)
			if rc >= maxReceive {
				// Send to DLQ and delete from main
				body := safeStr(m.Body)
				_, _ = sqsClient.SQS.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: &dlqURL, MessageBody: &body})
				_, _ = sqsClient.SQS.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: &mainURL, ReceiptHandle: m.ReceiptHandle})
				log.Println("moved to DLQ after retries. receiveCount=", rc, "err=", err)
				continue
			}

			// backoff by extending visibility timeout (simple)
			backoffSeconds := int32(math.Min(60, math.Pow(2, float64(rc)))) // 2s,4s,8s.. capped
			_, _ = sqsClient.SQS.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
				QueueUrl: &mainURL,
				ReceiptHandle: m.ReceiptHandle,
				VisibilityTimeout: backoffSeconds,
			})
			log.Println("temporary failure, will retry. receiveCount=", rc, "backoffSeconds=", backoffSeconds, "err=", err)
		}
	}
}

func handleMessage(ctx context.Context, repo persistence.Repo, policy config.Policy, provs []providers.Provider,
	bulk map[string]sem, cbs map[string]*gobreaker.CircuitBreaker, body *string, maxAttempts int) error {

	var ev OutboxEvent
	if err := json.Unmarshal([]byte(safeStr(body)), &ev); err != nil { return err }
	if ev.EventType != "PaymentRequested" || ev.PaymentID == "" { return errors.New("invalid event") }

	p, err := repo.FindByPaymentID(ctx, ev.PaymentID)
	if err != nil { return err }

	method := providers.Method(p.Method)
	providerChain := routeProviders(policy, string(method))
	if len(providerChain) == 0 { return errors.New("no providers for method") }

	// mark processing
	_ = repo.UpdateStatus(ctx, p.PaymentID, domain.StatusProcessing, p.Provider, nil, nil)

	// fallback chain
	attemptProvider := 0
	for attemptProvider < len(providerChain) {
		provName := providerChain[attemptProvider]
		prov := findProvider(provs, provName)
		if prov == nil { attemptProvider++; continue }

		s := bulk[provName]
		s.acquire()
		respAny, err := cbs[provName].Execute(func() (any, error) {
			var lastErr error
			operation := func() error {
				// call adapter
				ar := providers.AuthorizeRequest{
					PaymentID: p.PaymentID,
					AmountCents: p.AmountCents,
					Currency: p.Currency,
					// For PIX we store key/message in request in real impl.
					// In POC we reuse payer_name as pix key placeholder when PIX.
					PixKey: valueOr(p.PayerName, "email@merchant.com"),
					PixPayerMessage: "POC",
					PixExpiresSec: 900,
					CardHolderName: valueOr(p.PayerName, "Maria Silva"),
					CardNumber: "4111111111111111",
					CardExpMonth: 12,
					CardExpYear: 2030,
					CardCVV: "123",
					CardBrand: "VISA",
				}
				res, err := prov.Authorize(ctx, method, ar)
				if err != nil { lastErr = err; return err }
				// persist outcome
				status := domain.StatusPending
				switch res.Status {
				case "AUTHORIZED":
					status = domain.StatusPending
				case "PAID":
					status = domain.StatusPaid
				case "FAILED":
					status = domain.StatusFailed
				}
				_ = repo.UpdateProviderFields(ctx, p.PaymentID, status, persistence.ProviderFields{
					Provider: res.Provider,
					E2EID: res.E2EID,
					TXID: res.TxID,
					CardBrand: res.Brand,
					CardMaskedPAN: res.MaskedPAN,
					CardAuthCode: res.AuthCode,
				})
				return nil
			}

			b := backoff.NewExponentialBackOff()
			b.InitialInterval = 150 * time.Millisecond
			b.MaxInterval = 2 * time.Second
			b.MaxElapsedTime = 0
			return nil, backoff.Retry(backoff.WithMaxRetries(operation, uint64(maxAttempts-1)), b)
		}

		_, err := respAny, err
		if err != nil { return nil, err }
		return nil, nil
	})
		s.release()

		if err == nil {
			return nil
		}

		attemptProvider++
		if attemptProvider >= policy.Defaults.Fallback.MaxProviderHops {
			break
		}
	}

	_ = repo.UpdateStatus(ctx, p.PaymentID, domain.StatusFailed, p.Provider, nil, nil)
	return errors.New("all providers failed")
}

func routeProviders(policy config.Policy, method string) []string {
	for _, r := range policy.RoutingRules {
		if r.When.Method == method {
			return r.Providers
		}
	}
	return nil
}

func findProvider(list []providers.Provider, name string) providers.Provider {
	for _, p := range list {
		if p.Name() == name { return p }
	}
	return nil
}

func mustQueueURL(ctx context.Context, c *sqs.Client, name string) string {
	out, err := c.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: &name})
	if err != nil { log.Fatal("GetQueueUrl:", name, err) }
	return *out.QueueUrl
}

func approxReceiveCount(attrs map[string]string) int {
	v := attrs["ApproximateReceiveCount"]
	n, _ := strconv.Atoi(v)
	if n < 1 { n = 1 }
	return n
}

func safeStr(s *string) string {
	if s == nil { return "" }
	return *s
}

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" { return def }
	n, err := strconv.Atoi(v)
	if err != nil { return def }
	return n
}

func valueOr(p *string, def string) string {
	if p == nil || *p == "" { return def }
	return *p
}
