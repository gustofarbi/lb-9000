package utils

import (
	"github.com/redis/go-redis/v9"
	"lb-9000/lb-9000/internal/config"
)

func GetRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.StoreAddr,
		Username: cfg.StoreUsername,
		Password: cfg.StorePassword,
		DB:       cfg.StoreDB,
	})
}
