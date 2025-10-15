package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Client struct {
	base *redis.Client
}

func NewClient(address, password string, database int) (*Client, error) {
	if address == "" {
		return nil, errors.New("redis address is required")
	}

	base := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       database,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := base.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Client{base: base}, nil
}
