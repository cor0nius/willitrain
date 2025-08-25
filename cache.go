package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// This file defines the caching layer for the application. It includes a generic
// Cache interface and a Redis-backed implementation of that interface.

// Cache is an interface that defines the contract for a caching service.
// It provides basic operations like Set, Get, and Flush.
type Cache interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Flush(ctx context.Context) error
}

// RedisCache is a Redis implementation of the Cache interface.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new instance of RedisCache.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

// Set marshals a value to JSON and stores it in the Redis cache with a given key and expiration.
func (c *RedisCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	p, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, p, expiration).Err()
}

// Get retrieves a value from the Redis cache by its key.
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Flush deletes all keys from the current Redis database.
func (c *RedisCache) Flush(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}
