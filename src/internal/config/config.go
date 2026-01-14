package config

import "os"

type Config struct {
	DBURL       string
	SQSEndpoint string
	SQSQueue    string
	AWSRegion   string
	AccessKey   string
	SecretKey   string
	IngressPort string
	WebhookPort string
}

func Load() Config {
	return Config{
		DBURL:       getenv("DB_URL", "postgres://clickpay:clickpay@localhost:5432/clickpay?sslmode=disable"),
		SQSEndpoint: getenv("SQS_ENDPOINT", "http://localhost:9324"),
		SQSQueue:    getenv("SQS_QUEUE_NAME", "clickpay-payment-requested"),
		AWSRegion:   getenv("AWS_REGION", "us-east-1"),
		AccessKey:   getenv("AWS_ACCESS_KEY_ID", "x"),
		SecretKey:   getenv("AWS_SECRET_ACCESS_KEY", "x"),
		IngressPort: getenv("INGRESS_PORT", "8080"),
		WebhookPort: getenv("WEBHOOK_PORT", "8081"),
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" { return def }
	return v
}
