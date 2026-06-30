package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

// RedisCache caches ad records in Redis using the cache-aside pattern.
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache connects to Redis at addr and verifies the connection.
func NewRedisCache(ctx context.Context, addr string, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisCache{client: client, ttl: ttl}, nil
}

func key(id int64) string {
	return "ad:" + strconv.FormatInt(id, 10)
}

func (c *RedisCache) GetAd(ctx context.Context, id int64) (store.Ad, bool, error) {
	raw, err := c.client.Get(ctx, key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return store.Ad{}, false, nil
	}
	if err != nil {
		return store.Ad{}, false, fmt.Errorf("redis get: %w", err)
	}
	var ad store.Ad
	if err := json.Unmarshal(raw, &ad); err != nil {
		return store.Ad{}, false, fmt.Errorf("decode cached ad: %w", err)
	}
	return ad, true, nil
}

func (c *RedisCache) SetAd(ctx context.Context, ad store.Ad) error {
	raw, err := json.Marshal(ad)
	if err != nil {
		return fmt.Errorf("encode ad: %w", err)
	}
	if err := c.client.Set(ctx, key(ad.ID), raw, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}
