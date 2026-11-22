package redis

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(ctx context.Context, dsn string) (*redis.Client, error) {
	options := &redis.Options{
		DB:           0,
		PoolSize:     10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	if dsn != "" {
		parsed, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("parse redis DSN: %w", err)
		}
		options.Addr = parsed.Host
		if parsed.User != nil {
			options.Password, _ = parsed.User.Password()
		}
		if parsed.Scheme == "rediss" {
			options.TLSConfig = nil // will use default
		}
	} else {
		return nil, fmt.Errorf("redis DSN is empty")
	}

	client := redis.NewClient(options)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
