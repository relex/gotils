package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache[T any] interface {
	Get(key string) (*T, error)
	Set(key string, value T, expiration time.Duration) error
	SetNX(key string, value T, expiration time.Duration) (bool, error)
	Del(key string) error
	HealthCheck() error
}

type redisCache[T any] struct {
	client *redis.Client
}

var ctx = context.Background()

func NewRedisCache[T any](addr string, pwd string, db int, useTls bool) Cache[T] {
	var client *redis.Client
	if useTls {
		client = redis.NewClient(&redis.Options{
			Addr:      addr,
			Password:  pwd,
			TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			DB:        db,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pwd,
			DB:       db,
		})
	}
	return redisCache[T]{
		client: client,
	}
}

func (cache redisCache[T]) Get(key string) (*T, error) {
	val, err := cache.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		var result T
		err = json.Unmarshal([]byte(val), &result)
		if err != nil {
			return nil, err
		}
		return &result, nil
	}
}

func (cache redisCache[T]) Set(key string, value T, expiration time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = cache.client.Set(ctx, key, bytes, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

// SetNX sets the value of key `key` to `value` if the key does not exist.
func (cache redisCache[T]) SetNX(key string, value T, expiration time.Duration) (bool, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return cache.client.SetNX(ctx, key, bytes, expiration).Result()
}

func (cache redisCache[T]) Del(key string) error {
	err := cache.client.Del(ctx, key).Err()
	if err != nil {
		return err
	}
	return nil
}

func (cache redisCache[T]) HealthCheck() error {
	val, err := cache.client.Ping(ctx).Result()
	if err != nil {
		return err
	}
	if val != "PONG" {
		err := fmt.Errorf("received an invalid response to PING from redis")
		return err
	}
	return nil
}
