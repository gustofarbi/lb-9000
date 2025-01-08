package leader

import "github.com/redis/go-redis/v9"

type Elector struct {
	redis *redis.Client
}
