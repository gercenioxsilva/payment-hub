package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/you/clickpay-go-poc/internal/config"
	"github.com/you/clickpay-go-poc/internal/queue"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	c, err := queue.New(ctx, cfg.SQSEndpoint, cfg.AWSRegion, cfg.AccessKey, cfg.SecretKey)
	if err != nil { log.Fatal(err) }

	_, err = c.SQS.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: &cfg.SQSQueue})
	if err != nil { log.Fatal(err) }
	log.Println("queue created/ok:", cfg.SQSQueue)

	dlq := cfg.SQSQueue + "-dlq"
	_, err = c.SQS.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: &dlq})
	if err != nil { log.Fatal(err) }
	log.Println("queue created/ok:", dlq)

}
