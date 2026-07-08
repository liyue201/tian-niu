package redis_storage

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(addr, password string, db int) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisStorage{
		client: client,
	}
}

func (s *RedisStorage) Load(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

func (s *RedisStorage) Store(ctx context.Context, key string, value string) error {
	return s.client.Set(ctx, key, value, 0).Err()
}

func (s *RedisStorage) Close() error {
	return s.client.Close()
}
