package wallet

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/events"
	"github.com/zjoart/go-paystack-wallet/pkg/logger"
)

type WebhookWorker struct {
	Config      config.Config
	Repo        Repository
	RedisClient *events.RedisClient
}

func NewWebhookWorker(cfg config.Config, repo Repository, redisClient *events.RedisClient) *WebhookWorker {
	return &WebhookWorker{Config: cfg, Repo: repo, RedisClient: redisClient}
}

func (w *WebhookWorker) Start() {
	logger.Info("Starting webhook worker...")
	go w.processEvents()
}

func (w *WebhookWorker) processEvents() {
	for {

		result, err := w.RedisClient.Client.BLPop(context.Background(), 5*time.Second, events.WebhookQueue).Result()
		if err != nil {

			continue
		}

		eventData := []byte(result[1])
		var event events.WebhookEvent
		if err := json.Unmarshal(eventData, &event); err != nil {
			logger.Error("WebhookWorker: Failed to unmarshal event", logger.Fields{"error": err.Error(), "data": string(eventData)})
			w.moveToDLQ(eventData)
			continue
		}

		w.handleEvent(event, eventData)
	}
}

func (w *WebhookWorker) handleEvent(event events.WebhookEvent, rawData []byte) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := w.Repo.ProcessDeposit(event.Reference, event.Amount)
		if err == nil {
			logger.Info("WebhookWorker: Successfully processed deposit", logger.Fields{"reference": event.Reference})
			return
		}

		logger.Warn("WebhookWorker: Failed to process deposit, retrying", logger.Fields{
			"reference": event.Reference,
			"attempt":   i + 1,
			"error":     err.Error(),
		})
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	logger.Error("WebhookWorker: Max retries exhausted, moving to DLQ", logger.Fields{"reference": event.Reference})
	w.moveToDLQ(rawData)
}

func (w *WebhookWorker) moveToDLQ(data []byte) {
	if err := w.RedisClient.PushToDLQ(context.Background(), data); err != nil {
		logger.Error("Worker: Failed to push to DLQ", logger.Fields{"error": err.Error()})
	}
}
