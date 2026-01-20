package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
    client *redis.Client
    ctx    context.Context
}

type Config struct {
	Host string
	Port string
	Password string
	DB int
	PoolSize int
	MinIdleConns int
	MaxRetries int
	ReadTimeout time.Duration
	WriteTimeout time.Duration
	DialTimeout time.Duration
	ConnectTimeout time.Duration
}

func NewClient(config Config) (*Client, error) {
	address := fmt.Sprintf("%s:%s", config.Host, config.Port)

	options := &redis.Options{
		Addr: address,
		Password: config.Password,
		DB: config.DB,
		PoolSize: config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries: config.MaxRetries,
		DialTimeout: config.DialTimeout,
		ReadTimeout: config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}
    
    client := redis.NewClient(options)
    ctx := context.Background()
    
    // Test connection
    _, err := client.Ping(ctx).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }
    
    log.Printf("Connected to Redis at %s", address)
    
    return &Client{
        client: client,
        ctx: ctx,
    }, nil
}

func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Client) GetClient() *redis.Client {
    return c.client
}

func (c *Client) GetContext() context.Context {
    return c.ctx
}