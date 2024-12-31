package store

import (
	"iter"
	"lb-9000/lb-9000/internal/backend"
)

type Store interface {
	DebugStore

	Add(backend *backend.Backend)
	Remove(url string)
	AddRequests(url string, n int64)
	Iterate() iter.Seq[*backend.Backend]
}

type DebugStore interface {
	DebugPrint()
}
