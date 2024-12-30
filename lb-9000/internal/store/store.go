package store

import "lb-9000/lb-9000/internal/backend"

type Store interface {
	DebugStore

	Elect() *backend.Backend
	Add(backend *backend.Backend)
	Remove(url string)
	AddRequests(url string, n int64)
}

type DebugStore interface {
	DebugPrint()
}
