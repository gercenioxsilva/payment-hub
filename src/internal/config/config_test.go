package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("SQS_ENDPOINT", "")
	t.Setenv("SQS_QUEUE_NAME", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("INGRESS_PORT", "")
	t.Setenv("WEBHOOK_PORT", "")

	cfg := Load()

	if cfg.DBURL == "" {
		t.Fatal("expected default DBURL to be set")
	}
	if cfg.SQSEndpoint == "" || cfg.SQSQueue == "" {
		t.Fatal("expected default SQS settings to be set")
	}
	if cfg.IngressPort != "8080" {
		t.Fatalf("expected default ingress port 8080, got %s", cfg.IngressPort)
	}
	if cfg.WebhookPort != "8081" {
		t.Fatalf("expected default webhook port 8081, got %s", cfg.WebhookPort)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DB_URL", "db://override")
	t.Setenv("SQS_ENDPOINT", "http://sqs.local")
	t.Setenv("SQS_QUEUE_NAME", "queue-test")
	t.Setenv("AWS_REGION", "sa-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("INGRESS_PORT", "9090")
	t.Setenv("WEBHOOK_PORT", "9091")

	cfg := Load()

	if cfg.DBURL != "db://override" {
		t.Fatalf("expected DBURL override, got %s", cfg.DBURL)
	}
	if cfg.SQSEndpoint != "http://sqs.local" || cfg.SQSQueue != "queue-test" {
		t.Fatalf("expected SQS overrides, got endpoint=%s queue=%s", cfg.SQSEndpoint, cfg.SQSQueue)
	}
	if cfg.AWSRegion != "sa-east-1" || cfg.AccessKey != "key" || cfg.SecretKey != "secret" {
		t.Fatalf("expected AWS overrides, got region=%s key=%s secret=%s", cfg.AWSRegion, cfg.AccessKey, cfg.SecretKey)
	}
	if cfg.IngressPort != "9090" || cfg.WebhookPort != "9091" {
		t.Fatalf("expected port overrides, got ingress=%s webhook=%s", cfg.IngressPort, cfg.WebhookPort)
	}
}
