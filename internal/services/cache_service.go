package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mat/arcapi/internal/config"
)

type CacheService struct {
	client *redis.Client
	ctx    context.Context
}

func NewCacheService(cfg *config.Config) (*CacheService, error) {
	var redisOpts *redis.Options

	// Parse Redis URL if provided, otherwise use separate config
	if cfg.RedisURL != "" {
		parsedURL, err := url.Parse(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid Redis URL: %w", err)
		}

		// Extract password from URL if present (format: redis://password@host:port)
		password := ""
		if parsedURL.User != nil {
			password, _ = parsedURL.User.Password()
			if password == "" {
				password = parsedURL.User.Username() // Sometimes password is provided as username
			}
		}

		// Extract host and port
		host := parsedURL.Hostname()
		port := parsedURL.Port()
		if port == "" {
			port = "6379" // Default Redis port
		}
		addr := fmt.Sprintf("%s:%s", host, port)

		redisOpts = &redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		}
	} else {
		// Fallback to separate config
		redisOpts = &redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       0,
		}
	}

	// Try to parse DB number from URL path if present (redis://host:port/0)
	if cfg.RedisURL != "" {
		parsedURL, _ := url.Parse(cfg.RedisURL)
		if parsedURL != nil && parsedURL.Path != "" {
			if dbNum, err := strconv.Atoi(strings.TrimPrefix(parsedURL.Path, "/")); err == nil {
				redisOpts.DB = dbNum
			}
		}
	}

	client := redis.NewClient(redisOpts)

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

// Client returns the underlying Redis client (for advanced operations like rate limiting)
func (s *CacheService) Client() *redis.Client {
	return s.client
}

// Context returns the context used by the cache service
func (s *CacheService) Context() context.Context {
	return s.ctx
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
