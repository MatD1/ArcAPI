package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mat/arcapi/internal/config"
)

type CacheService struct {
	client *redis.Client
	ctx    context.Context
}

func NewCacheService(cfg *config.Config) (*CacheService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheService{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *CacheService) Get(key string) ([]byte, error) {
	val, err := s.client.Get(s.ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (s *CacheService) Set(key string, value []byte, ttl time.Duration) error {
	return s.client.Set(s.ctx, key, value, ttl).Err()
}

func (s *CacheService) GetJSON(key string, dest interface{}) error {
	val, err := s.Get(key)
	if err != nil {
		return err
	}
	if val == nil {
		return nil
	}
	return json.Unmarshal(val, dest)
}

func (s *CacheService) SetJSON(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.Set(key, data, ttl)
}

func (s *CacheService) Delete(key string) error {
	return s.client.Del(s.ctx, key).Err()
}

func (s *CacheService) DeletePattern(pattern string) error {
	keys, err := s.client.Keys(s.ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.client.Del(s.ctx, keys...).Err()
}

func (s *CacheService) Close() error {
	return s.client.Close()
}

// Cache key helpers
func APIKeyCacheKey(hash string) string {
	return fmt.Sprintf("api_key:%s", hash)
}

func JWTCacheKey(hash string) string {
	return fmt.Sprintf("jwt:%s", hash)
}

func DataCacheKey(entity, key string) string {
	return fmt.Sprintf("data:%s:%s", entity, key)
}
