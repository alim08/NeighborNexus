package database

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisClient wraps the Redis client
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(addr, password string, db int) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisClient{
		Client: client,
	}
}

// Ping tests the Redis connection
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Set sets a key-value pair with optional expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Del deletes a key
func (r *RedisClient) Del(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.Client.Exists(ctx, key).Result()
	return result > 0, err
}

// Incr increments a counter
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

// Expire sets expiration for a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// Rate limiting functions
func (r *RedisClient) IsRateLimited(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	current, err := r.Incr(ctx, key)
	if err != nil {
		return true, err
	}

	if current == 1 {
		r.Expire(ctx, key, window)
	}

	return current > int64(limit), nil
}

// Cache functions
func (r *RedisClient) SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.Set(ctx, "cache:"+key, value, ttl)
}

func (r *RedisClient) GetCache(ctx context.Context, key string) (string, error) {
	return r.Get(ctx, "cache:"+key)
}

// Job queue functions
func (r *RedisClient) EnqueueJob(ctx context.Context, queue string, job interface{}) error {
	return r.Client.LPush(ctx, "queue:"+queue, job).Err()
}

func (r *RedisClient) DequeueJob(ctx context.Context, queue string) (string, error) {
	result, err := r.Client.BRPop(ctx, 0, "queue:"+queue).Result()
	if err != nil {
		return "", err
	}
	if len(result) < 2 {
		return "", nil
	}
	return result[1], nil
}

// WebSocket session management
func (r *RedisClient) AddWebSocketSession(ctx context.Context, userID, sessionID string) error {
	return r.Set(ctx, "ws:"+userID, sessionID, 24*time.Hour)
}

func (r *RedisClient) GetWebSocketSession(ctx context.Context, userID string) (string, error) {
	return r.Get(ctx, "ws:"+userID)
}

func (r *RedisClient) RemoveWebSocketSession(ctx context.Context, userID string) error {
	return r.Del(ctx, "ws:"+userID)
} 