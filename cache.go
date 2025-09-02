package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// This file implements the application's caching layer, which is crucial for
// performance and scalability. It provides a fast, in-memory storage for frequently
// accessed data, reducing latency for users and minimizing load on the database and
// external APIs. The implementation uses a generic interface to decouple the
// application from the specific caching technology (Redis).

// Cache is a generic interface for a key-value cache.
// Defining an interface for the cache allows the underlying implementation to be
// swapped out without affecting the rest of the application. It also simplifies
// testing by allowing a mock cache to be used in place of a real one.
type Cache interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Flush(ctx context.Context) error
}

// RedisCache is a Redis-backed implementation of the Cache interface.
// It uses a redis.Client to interact with the Redis server.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates and returns a new instance of RedisCache.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

// Set serializes the given value to JSON and stores it in the Redis cache.
// This approach allows complex data structures to be cached as simple strings.
// An expiration is set to ensure that stale data is automatically evicted.
func (c *RedisCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	p, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, p, expiration).Err()
}

// Get retrieves an item from the Redis cache by its key.
// The returned value is a raw string, which the caller is responsible for
// deserializing back into a Go struct.
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Flush removes all keys from the current Redis database.
// This is primarily used in development and testing to reset the application's state.
func (c *RedisCache) Flush(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// ConnectCache initializes the cache connection.
// It currently supports only Redis, but the interface allows for future extensions.
func (cfg *apiConfig) ConnectCache() error {
	return cfg.ConnectRedis()
}

// ConnectRedis initializes the Redis client and verifies the connection.
// If successful, it assigns a RedisCache instance to the apiConfig's cache field.
func (cfg *apiConfig) ConnectRedis() error {
	opt, err := redis.ParseURL(cfg.redisURL)
	if err != nil {
		cfg.logger.Error("could not parse Redis URL", "error", err)
		return err
	}
	redisClient := cfg.newCacheClientFunc(opt)
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		cfg.logger.Error("could not connect to Redis", "error", err)
		return err
	}
	cfg.cache = NewRedisCache(redisClient)
	cfg.logger.Debug("connected to Redis cache")
	return nil
}