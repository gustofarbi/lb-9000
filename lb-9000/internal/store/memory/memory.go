package memory

import (
	"fmt"
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

func (p *Map) Add(backend *backend.Backend) {
	url := backend.URL()
	name := backend.Name()

	p.logger.Info("adding", "url", url, "name", name)

	p.inner[url] = backend
}

func (p *Map) Remove(url string) {
	if url == "" {
		return
	}

	p.logger.Info(fmt.Sprintf("pod '%s' deleted", url))
	delete(p.inner, url)
}

func (p *Map) Elect() *backend.Backend {
	var (
		minCount int64 = math.MaxInt64
		minPod   *backend.Backend
	)

	p.lock.Lock()
	defer p.lock.Unlock()

	for _, pod := range p.inner {
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

func (p *Map) AddRequests(url string, delta int64) {
	if url == "" {
		return
	}

	instance, ok := p.inner[url]
	if !ok {
		p.logger.Info("could not find backend", "url", url)
		return
	}

	instance.AddRequests(delta)
}

func (p *Map) DebugPrint() {
	for _, pod := range p.inner {
		p.logger.Info(fmt.Sprintf("pod '%s' has '%d' requests", pod.URL(), pod.Count()))
	}
}
