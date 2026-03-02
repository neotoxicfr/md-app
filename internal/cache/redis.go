package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 10 * time.Minute

// Client wraps a Redis connection for caching rendered HTML.
type Client struct {
	rdb *redis.Client
}

// New parses the Redis URL and returns a connected Client.
func New(url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Client{rdb: rdb}, nil
}

// Set stores a value with a default TTL.
func (c *Client) Set(ctx context.Context, key, value string) error {
	return c.rdb.Set(ctx, key, value, defaultTTL).Err()
}

// Get retrieves a value, returns ("", nil) on cache miss.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Delete invalidates a cache entry.
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// Close shuts down the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
