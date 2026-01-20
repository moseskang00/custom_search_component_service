package cache

import (
	"fmt"
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	redisClient &redis.client
	ctx context.Context
	prefix string
}	

func NewCache(client *redis.Client, prefix string) *Cache {
	return &Cache{
		redisClient: client,
		ctx: context.Background(),
		prefix: prefix,
	}
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) error {
	var data interface{} = value

	switch v := value.(type) {
	case string:
		data = v
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value to JSON: %w", err)
		}
		data = jsonData
	}

	fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
	return c.redisClient.Set(c.ctx, fullKey, data, ttl).Err()
}

func (c *Cache) Get(key string) (string, error) {
	fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
	return c.redisClient.Get(c.ctx, fullKey).Result()
}

func (c *Cache) GetJSON(key string, v interface{}) error {
	fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
	jsonData, err := c.redisClient.Get(c.ctx, fullKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get value from Redis: %w", err)
	}
	return json.Unmarshal([]byte(jsonData), v)
}

func (c *Cache) Delete(key string) error {
	fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
	return c.redisClient.Del(c.ctx, fullKey).Err()
}

func (c *Cache) Exists(key string) (bool, error) {
    fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
    result, err := c.redisClient.Exists(c.ctx, fullKey).Result()
    return result > 0, err
}

func (c *Cache) Increment(key string) (int64, error) {
    fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
    return c.redisClient.Incr(c.ctx, fullKey).Result()
}

func (c *Cache) GetTTL(key string) (time.Duration, error) {
    fullKey := fmt.Sprintf("%s:%s", c.prefix, key)
    return c.redisClient.TTL(c.ctx, fullKey).Result()
}

// Keys gets all keys matching pattern --> might be useful for later..
func (c *Cache) Keys(pattern string) ([]string, error) {
    fullPattern := fmt.Sprintf("%s:%s", c.prefix, pattern)
    return c.redisClient.Keys(c.ctx, fullPattern).Result()
}

// FlushAll clears all cache
func (c *Cache) FlushAll() error {
    return c.redisClient.FlushAll(c.ctx).Err()
}