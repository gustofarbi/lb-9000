package memory

import (
	"context"
	"fmt"
	"iter"
	"lb-9000/lb-9000/internal/backend"
	"log/slog"
	"sync"
)

type Map struct {
	inner  map[string]*backend.Backend
	logger *slog.Logger
	lock   *sync.Mutex
}

func NewMemoryStore(logger *slog.Logger) *Map {
	if logger == nil {
		logger = slog.Default()
	}

	return &Map{
		inner:  make(map[string]*backend.Backend),
		logger: logger,
		lock:   &sync.Mutex{},
	}
}

func (m *Map) Add(_ context.Context, backend *backend.Backend) error {
	url := backend.URL()
	name := backend.Name()

	m.logger.Info("adding", "url", url, "name", name)

	m.inner[url] = backend

	return nil
}

func (m *Map) Remove(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	m.logger.Info(fmt.Sprintf("pod '%s' deleted", id))
	delete(m.inner, id)

	return nil
}

func (m *Map) AddRequests(_ context.Context, id string, delta int64) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	instance, ok := m.inner[id]
	if !ok {
		return fmt.Errorf("could not find backend")
	}

	instance.AddRequests(delta)

	return nil
}

func (m *Map) Iterate(context.Context) (iter.Seq[*backend.Backend], error) {
	return func(yield func(*backend.Backend) bool) {
		m.lock.Lock()
		defer m.lock.Unlock()

		for _, pod := range m.inner {
			if !yield(pod) {
				return
			}
		}
	}, nil
}
