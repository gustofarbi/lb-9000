package store

import (
	"context"
	"iter"
	"lb-9000/lb-9000/internal/backend"
	"lb-9000/lb-9000/internal/config"
	"lb-9000/lb-9000/internal/store/memory"
	"lb-9000/lb-9000/internal/store/redis"
	"log/slog"
)

type Store interface {
	Add(ctx context.Context, backend *backend.Backend) error
	Remove(ctx context.Context, id string) error
	AddRequests(ctx context.Context, id string, n int64) error
	Iterate(ctx context.Context) (iter.Seq[*backend.Backend], error)
	All(ctx context.Context) ([]*backend.Backend, error)
}

func Get(config *config.Config, logger *slog.Logger) Store {
	switch config.Store.Type {
	case "memory":
		return memory.New(logger)
	case "redis":
		return redis.New(logger, config)
	default:
		panic("unknown store type")
	}
}
