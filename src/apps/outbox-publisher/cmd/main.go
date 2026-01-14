package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/you/clickpay-go-poc/internal/config"
	"github.com/you/clickpay-go-poc/internal/persistence"
	"github.com/you/clickpay-go-poc/internal/queue"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := persistence.Connect(ctx, cfg.DBURL)
	if err != nil { log.Fatal(err) }
	repo := persistence.Repo{DB: db}

	sqsClient, err := queue.New(ctx, cfg.SQSEndpoint, cfg.AWSRegion, cfg.AccessKey, cfg.SecretKey)
	if err != nil { log.Fatal(err) }

	qurlOut, err := sqsClient.SQS.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: &cfg.SQSQueue})
	if err != nil { log.Fatal(err) }
	qurl := *qurlOut.QueueUrl

	log.Println("outbox-publisher started, queue:", qurl)

	ticker := time.NewTicker(500 * time.Millisecond)
	for range ticker.C {
		items, err := repo.FetchUnpublishedOutbox(ctx, 50)
		if err != nil { log.Println("fetch outbox:", err); continue }
		for _, it := range items {
			_, err := sqsClient.SQS.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: &qurl, MessageBody: &it.Payload})
			if err != nil { log.Println("send sqs:", err); continue }
			_ = repo.MarkOutboxPublished(ctx, it.ID)
		}
	}
}
