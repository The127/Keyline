package utils

import (
	"Keyline/internal/config"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.C.Redis.Host, config.C.Redis.Port),
		Username: config.C.Redis.Username,
		Password: config.C.Redis.Password,
		DB:       config.C.Redis.Database,
	})
}
