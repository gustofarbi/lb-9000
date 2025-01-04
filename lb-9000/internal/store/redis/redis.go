package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"iter"
	"lb-9000/lb-9000/internal/backend"
	"lb-9000/lb-9000/internal/config"
	"log/slog"
	"strings"
)

const cacheTag = "backends"

func New(logger *slog.Logger, cfg *config.Config) *Redis {
	return &Redis{
		redis: redis.NewClient(
			&redis.Options{
				Addr:     cfg.Store.Addr,
				Username: cfg.Store.Username,
				Password: cfg.Store.Password,
				DB:       cfg.Store.DB,
			},
		),
		logger: logger,
	}
}

type Redis struct {
	redis  *redis.Client
	logger *slog.Logger
}

func (r *Redis) Iterate(ctx context.Context) (iter.Seq[*backend.Backend], error) {
	keys, err := r.redis.SMembers(ctx, cacheTag).Result()
	if err != nil {
		return nil, fmt.Errorf("getting keys by tag: %w", err)
	}

	backends, err := r.redis.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("getting backends by keys (%s): %w", strings.Join(keys, ", "), err)
	}

	result := make([]*backend.Backend, 0, len(backends))

	// todo test this, does it even work???
	for _, backendCandidate := range backends {
		b := backendCandidate.(backend.Backend)
		result = append(result, &b)
	}

	return func(yield func(*backend.Backend) bool) {
		for _, b := range result {
			if !yield(b) {
				return
			}
		}
	}, nil
}

func (r *Redis) Add(ctx context.Context, backend *backend.Backend) error {
	pipe := r.redis.TxPipeline()

	url := backend.URL()

	pipe.SAdd(ctx, cacheTag, url)
	pipe.Set(ctx, url, backend, 0)
	//todo pipe.JSONSet()

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("executing pipeline: %w", err)
	}

	return nil
}

func (r *Redis) Remove(ctx context.Context, id string) error {
	if _, err := r.redis.Del(ctx, id).Result(); err != nil {
		return fmt.Errorf("deleting backend '%s': %w", id, err)
	}

	return nil
}

func (r *Redis) AddRequests(ctx context.Context, id string, n int64) error {
	pipe := r.redis.TxPipeline()

	result, err := pipe.MGet(ctx, id).Result()
	if err != nil {
		return fmt.Errorf("getting backend '%s': %w", id, err)
	}

	if len(result) == 0 {
		// backend could be deleted here
		r.logger.Debug("backend not found", "id", id)
		// todo close the pipe???
		return nil
	}

	b, ok := result[0].(*backend.Backend)
	if !ok {
		return fmt.Errorf("converting result to a backend object: '%+v'", result[0])
	}

	b.AddRequests(n)

	if _, err = pipe.Set(ctx, id, b, 0).Result(); err != nil {
		return fmt.Errorf("saving backend '%s': %w", id, err)
	}

	return nil
}
