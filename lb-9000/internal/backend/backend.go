package backend

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

type Backend struct {
	id    string
	name  string
	count *atomic.Int64
}

type innerBackend struct {
	URL   string `json:"id"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

func (p *Backend) URL() string {
	return p.id
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

func (p *Backend) MarshalBinary() (data []byte, err error) {
	return json.Marshal(innerBackend{
		URL:   p.id,
		Name:  p.name,
		Count: p.count.Load(),
	})
}

func (p *Backend) UnmarshalBinary(data []byte) error {
	var inner innerBackend
	if err := json.Unmarshal(data, &inner); err != nil {
		return fmt.Errorf("unmarshalling backend: %w", err)
	}

	p.id = inner.URL
	p.name = inner.Name
	p.count = new(atomic.Int64)
	p.count.Store(inner.Count)

	return nil
}

func NewBackend(id, name string) *Backend {
	return &Backend{
		id:    id,
		name:  name,
		count: new(atomic.Int64),
	}
}
