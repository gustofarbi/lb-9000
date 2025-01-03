package store

import (
	"context"
	"iter"
	"lb-9000/lb-9000/internal/backend"
)

type Store interface {
	Add(ctx context.Context, backend *backend.Backend) error
	Remove(ctx context.Context, url string) error
	AddRequests(ctx context.Context, url string, n int64) error
	Iterate(ctx context.Context) (iter.Seq[*backend.Backend], error)
}
