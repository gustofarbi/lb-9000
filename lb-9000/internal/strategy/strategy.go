package strategy

import (
	"context"
	"fmt"
	"lb-9000/lb-9000/internal/backend"
	"lb-9000/lb-9000/internal/store"
	"math"
	"sync/atomic"
)

type Strategy interface {
	Elect(ctx context.Context, store store.Store) (*backend.Backend, error)
}

type roundRobin struct {
	currentIndex uint64
}

func RoundRobin() Strategy {
	return roundRobin{}
}

func (r roundRobin) Elect(ctx context.Context, store store.Store) (*backend.Backend, error) {
	backends, err := store.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting all backends: %w", err)
	}

	return backends[atomic.AddUint64(&r.currentIndex, 1)%uint64(len(backends))], nil
}

func FillHoles() Strategy {
	return fillHolesStrategy{}
}

type fillHolesStrategy struct{}

func (f fillHolesStrategy) Elect(ctx context.Context, store store.Store) (*backend.Backend, error) {
	var (
		minCount   int64 = math.MaxInt64
		minBackend *backend.Backend
	)

	iterator, err := store.Iterate(ctx)
	if err != nil {
		return nil, fmt.Errorf("iterating backends: %w", err)
	}

	for instance := range iterator {
		if count := instance.Count(); count < minCount {
			minCount = count
			minBackend = instance
		}
		if minCount == 0 {
			break
		}
	}

	return minBackend, nil
}
