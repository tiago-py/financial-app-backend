package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func Open(ctx context.Context, addr, password string, db int) (*Cache, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("conectar redis: %w", err)
	}
	return &Cache{client: client}, nil
}

func (c *Cache) Close() error { return c.client.Close() }

func (c *Cache) GetJSON(ctx context.Context, key string, destination any) (bool, error) {
	value, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(value, destination); err != nil {
		_ = c.client.Del(ctx, key).Err()
		return false, nil
	}
	return true, nil
}

func (c *Cache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, encoded, ttl).Err()
}

func (c *Cache) InvalidateUser(ctx context.Context, userID string) {
	pattern := "user:" + userID + ":*"
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = c.client.Del(ctx, keys...).Err()
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}
