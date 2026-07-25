package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/apolinario0x21/effective-octo-bassoon/internal/config"
)

// Client é um wrapper fino sobre o cliente Redis, mantendo o TTL padrão
// usado pela aplicação.
type Client struct {
	rdb *redis.Client
	ttl time.Duration
}

// Connect abre a conexão com o Redis e valida com um PING.
func Connect(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("pinging redis at %s: %w", cfg.Addr, err)
	}

	return &Client{rdb: rdb, ttl: cfg.TTL}, nil
}

// Get devolve o valor bruto de uma chave. Retorna (nil, nil) quando a chave
// não existe (cache miss).
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

// Set grava um valor usando o TTL padrão configurado.
func (c *Client) Set(ctx context.Context, key string, value []byte) error {
	return c.rdb.Set(ctx, key, value, c.ttl).Err()
}

// Delete remove uma ou mais chaves do cache.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// Close encerra a conexão com o Redis.
func (c *Client) Close() error {
	return c.rdb.Close()
}
