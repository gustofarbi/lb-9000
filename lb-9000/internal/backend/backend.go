package backend

import (
	"sync/atomic"
)

type Backend struct {
	url   string
	name  string
	count *atomic.Int64
}

func (p *Backend) URL() string {
	return p.url
}

func (p *Backend) Name() string {
	return p.name
}

func (p *Backend) Count() int64 {
	return p.count.Load()
}

func (p *Backend) AddRequests(n int64) {
	newCount := max(0, p.count.Load()+n)
	p.count.Store(newCount)
}

func NewBackend(ip, name string) *Backend {
	return &Backend{
		url:   ip,
		name:  name,
		count: new(atomic.Int64),
	}
}
