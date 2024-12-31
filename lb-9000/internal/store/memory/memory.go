package memory

import (
	"fmt"
	"iter"
	"lb-9000/lb-9000/internal/backend"
	"log/slog"
	"math"
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

func (m *Map) Add(backend *backend.Backend) {
	url := backend.URL()
	name := backend.Name()

	m.logger.Info("adding", "url", url, "name", name)

	m.inner[url] = backend
}

func (m *Map) Remove(url string) {
	if url == "" {
		return
	}

	m.logger.Info(fmt.Sprintf("pod '%s' deleted", url))
	delete(m.inner, url)
}

func (m *Map) Elect() *backend.Backend {
	var (
		minCount int64 = math.MaxInt64
		minPod   *backend.Backend
	)

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, pod := range m.inner {
		if count := pod.Count(); count < minCount {
			minCount = count
			minPod = pod
		}
		if minCount == 0 {
			break
		}
	}

	return minPod
}

func (m *Map) AddRequests(url string, delta int64) {
	if url == "" {
		return
	}

	instance, ok := m.inner[url]
	if !ok {
		m.logger.Info("could not find backend", "url", url)
		return
	}

	instance.AddRequests(delta)
}

func (m *Map) DebugPrint() {
	for _, pod := range m.inner {
		m.logger.Info(fmt.Sprintf("pod '%s' has '%d' requests", pod.URL(), pod.Count()))
	}
}

func (m *Map) Iterate() iter.Seq[*backend.Backend] {
	return func(yield func(*backend.Backend) bool) {
		m.lock.Lock()
		defer m.lock.Unlock()

		for _, pod := range m.inner {
			if !yield(pod) {
				return
			}
		}
	}
}
