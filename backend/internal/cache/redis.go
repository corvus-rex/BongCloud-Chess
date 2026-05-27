package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/bongcloud/chess/internal/config"
)

// RedisClient wraps go-redis/v9 and exposes helpers used throughout the app.
// Domain packages receive this via dependency injection rather than calling
// redis.NewClient directly, keeping the cache layer swappable.
type RedisClient struct {
	Client *redis.Client
	log    *zap.Logger
}

// NewRedisClient creates a Redis client, verifies connectivity with PING,
// and returns the ready-to-use wrapper. Returns an error on connection failure.
func NewRedisClient(cfg config.RedisConfig, log *zap.Logger) (*RedisClient, error) {
	log.Info("connecting to Redis", zap.String("addr", cfg.Addr()))

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,

		// Retry settings for transient failures
		MaxRetries:      3,
		MinRetryBackoff: 8,
		MaxRetryBackoff: 512,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Info("Redis connected",
		zap.String("addr", cfg.Addr()),
		zap.Int("pool_size", cfg.PoolSize),
		zap.Int("db", cfg.DB),
	)

	return &RedisClient{
		Client: rdb,
		log:    log,
	}, nil
}

func (r *RedisClient) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

func (r *RedisClient) Close() error {
	r.log.Info("closing Redis connection pool")
	if err := r.Client.Close(); err != nil {
		return fmt.Errorf("redis close: %w", err)
	}
	r.log.Info("Redis disconnected")
	return nil
}

func (r *RedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.Client.Subscribe(ctx, channels...)
}

func (r *RedisClient) Publish(ctx context.Context, channel string, payload interface{}) error {
	return r.Client.Publish(ctx, channel, payload).Err()
}

func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.Client.Pipeline()
}

func (r *RedisClient) TxPipeline() redis.Pipeliner {
	return r.Client.TxPipeline()
}