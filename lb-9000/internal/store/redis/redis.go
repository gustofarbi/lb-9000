package redis

import (
	"lb-9000/lb-9000/internal/backend"
)

type Redis struct {
}

func (r Redis) DebugPrint() {
	//TODO implement me
	panic("implement me")
}

func (r Redis) Elect() *backend.Backend {
	//TODO implement me
	panic("implement me")
}

func (r Redis) Add(backend *backend.Backend) {
	//TODO implement me
	panic("implement me")
}

func (r Redis) Remove(url string) {
	//TODO implement me
	panic("implement me")
}

func (r Redis) AddRequests(url string, n int64) {
	//TODO implement me
	panic("implement me")
}
