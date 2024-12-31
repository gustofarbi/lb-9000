package strategy

import (
	"lb-9000/lb-9000/internal/backend"
	"lb-9000/lb-9000/internal/store"
	"math"
)

type Strategy interface {
	Elect(store store.Store) *backend.Backend
}

func FillHoles() Strategy {
	return fillHolesStrategy{}
}

type fillHolesStrategy struct{}

func (f fillHolesStrategy) Elect(store store.Store) *backend.Backend {
	var (
		minCount   int64 = math.MaxInt64
		minBackend *backend.Backend
	)

	for instance := range store.Iterate() {
		if count := instance.Count(); count < minCount {
			minCount = count
			minBackend = instance
		}
		if minCount == 0 {
			break
		}
	}

	return minBackend
}
