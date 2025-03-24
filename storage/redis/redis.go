package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client  *redis.Client
	prefix  string
	ttl     time.Duration
	lockTTL time.Duration
}

type Option func(*RedisStorage)

func WithPrefix(prefix string) Option {
	return func(r *RedisStorage) {
		r.prefix = prefix
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(r *RedisStorage) {
		r.ttl = ttl
	}
}

func WithLockTTL(ttl time.Duration) Option {
	return func(r *RedisStorage) {
		r.lockTTL = ttl
	}
}

func NewRedisStorage(client *redis.Client, opts ...Option) *RedisStorage {
	r := &RedisStorage{
		client:  client,
		prefix:  "fsm",
		ttl:     0,
		lockTTL: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *RedisStorage) key(id string) string {
	return fmt.Sprintf("%s:%s", r.prefix, id)
}

func (r *RedisStorage) lockKey(id string) string {
	return fmt.Sprintf("%s:lock:%s", r.prefix, id)
}

func (r *RedisStorage) GetState(ctx context.Context, entityID string) (string, error) {
	key := r.key(entityID)
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("redis: state not found for ID '%s'", entityID)
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

func (r *RedisStorage) SetState(ctx context.Context, entityID, state string) error {
	key := r.key(entityID)
	if r.ttl > 0 {
		return r.client.Set(ctx, key, state, r.ttl).Err()
	}
	return r.client.Set(ctx, key, state, 0).Err()
}

func (r *RedisStorage) Lock(ctx context.Context, entityID string) (func(), error) {
	key := r.lockKey(entityID)
	ok, err := r.client.SetNX(ctx, key, "locked", r.lockTTL).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("fsm: unable to acquire lock")
	}

	unlock := func() {
		_ = r.client.Del(ctx, key).Err()
	}

	return unlock, nil
}
