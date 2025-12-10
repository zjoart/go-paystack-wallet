package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/logger"
)

const (
	WebhookQueue = "webhook_events"
	FailedQueue  = "failed_webhook_events"
)

type RedisClient struct {
	Client *redis.Client
}

type WebhookEvent struct {
	Event     string    `json:"event"`
	Reference string    `json:"reference"`
	Status    string    `json:"status"`
	Amount    int64     `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

func NewRedisClient(cfg config.Config) *RedisClient {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("Failed to parse Redis url", logger.Fields{"error": err.Error(), "url": cfg.RedisURL})
		opt = &redis.Options{
			Addr:     cfg.RedisURL,
			Password: cfg.RedisPassword,
			DB:       0,
		}
	}

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		logger.Error("Failed to connect to Redis", logger.Fields{"error": err.Error(), "url": cfg.RedisURL})

	} else {
		logger.Info("Connected to Redis", logger.Fields{"url": cfg.RedisURL})
	}

	return &RedisClient{Client: rdb}
}

func (r *RedisClient) PublishEvent(ctx context.Context, event WebhookEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	if err := r.Client.RPush(ctx, WebhookQueue, data).Err(); err != nil {
		return fmt.Errorf("failed to push event to redis: %v", err)
	}

	return nil
}

func (r *RedisClient) PushToDLQ(ctx context.Context, data []byte) error {
	if err := r.Client.RPush(ctx, FailedQueue, data).Err(); err != nil {
		return fmt.Errorf("failed to push event to DLQ: %v", err)
	}
	return nil
}
