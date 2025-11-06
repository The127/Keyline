package keyValue

import (
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"Keyline/internal/middlewares"
	"context"
	"errors"
	"fmt"
	"github.com/The127/ioc"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

type Options struct {
	Expiration time.Duration
}

type Option func(*Options)

func WithExpiration(expiration time.Duration) Option {
	return func(o *Options) {
		o.Expiration = expiration
	}
}

type Store interface {
	Set(ctx context.Context, key string, value string, opts ...Option) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

func NewMemoryStore() Store {
	return &memoryStore{
		data: make(map[string]memoryStoreItem),
	}
}

type memoryStoreItem struct {
	value      string
	expiration time.Time
}

func (m *memoryStoreItem) IsExpired(now time.Time) bool {
	if m.expiration.IsZero() {
		return false
	}

	return m.expiration.Before(now)
}

type memoryStore struct {
	data map[string]memoryStoreItem
	mu   sync.RWMutex
}

func (m *memoryStore) Set(ctx context.Context, key string, value string, opts ...Option) error {
	scope := middlewares.GetScope(ctx)
	clockService := ioc.GetDependency[clock.Service](scope)

	item := memoryStoreItem{
		value: value,
	}

	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.Expiration != 0 {
		item.expiration = clockService.Now().Add(options.Expiration)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = item
	return nil
}

func (m *memoryStore) Get(ctx context.Context, key string) (string, error) {
	scope := middlewares.GetScope(ctx)
	clockService := ioc.GetDependency[clock.Service](scope)

	m.mu.RLock()
	item, ok := m.data[key]
	if !ok {
		m.mu.RUnlock()
		return "", ErrNotFound
	}
	m.mu.RUnlock()

	if item.IsExpired(clockService.Now()) {
		m.mu.Lock()
		itemBeforeDeletion := m.data[key]
		if itemBeforeDeletion.IsExpired(clockService.Now()) {
			delete(m.data, key)
		}
		m.mu.Unlock()
		return "", ErrNotFound
	}

	return item.value, nil
}

func (m *memoryStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func NewRedisStore() Store {
	return &redisKvStore{}
}

type redisKvStore struct {
}

func (r *redisKvStore) Set(ctx context.Context, key string, value string, opts ...Option) error {
	client := newRedisClient()
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return client.Set(ctx, key, value, options.Expiration).Err()
}

func (r *redisKvStore) Get(ctx context.Context, key string) (string, error) {
	client := newRedisClient()
	result, err := client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	return result, err
}

func (r *redisKvStore) Delete(ctx context.Context, key string) error {
	client := newRedisClient()
	err := client.Del(ctx, key).Err()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.C.Cache.Redis.Host, config.C.Cache.Redis.Port),
		Username: config.C.Cache.Redis.Username,
		Password: config.C.Cache.Redis.Password,
		DB:       config.C.Cache.Redis.Database,
	})
}
