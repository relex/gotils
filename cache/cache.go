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

func (cache redisCache[T]) Del(keys string) error {
	val, err := cache.client.Del(ctx, keys).Result()
	if err != nil {
		return err
	}
	if val == 0 {
		err := fmt.Errorf("nothing to delete with key(s): %s", keys)
		return err
	}
	return nil
}
