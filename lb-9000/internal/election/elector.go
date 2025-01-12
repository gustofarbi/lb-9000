package election

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	leaderKey = "proxy_leader"
)

type Status uint32

const (
	StatusFollower Status = iota
	StatusLeader
)

type Elector struct {
	id       string
	logger   *slog.Logger
	redis    *redis.Client
	status   *atomic.Uint32
	shutdown bool
	lockTTL  time.Duration
}

func NewElector(
	id string,
	logger *slog.Logger,
	redis *redis.Client,
	lockTTL time.Duration,
) *Elector {
	return &Elector{
		id:      id,
		logger:  logger,
		redis:   redis,
		status:  &atomic.Uint32{},
		lockTTL: lockTTL,
	}
}

func (e *Elector) IsLeader() bool {
	return e.status.Load() == uint32(StatusLeader)
}

func (e *Elector) Loop() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-stopCh
		e.shutdown = true
		cancel()
	}()

	for {
		if e.shutdown {
			e.logger.Info("shutting down")
			return
		}

		if e.tryBecomeLeader(ctx) {
			e.logger.Info("i am the leader", "instanceId", e.id)
			e.status.Store(uint32(StatusLeader))
			if err := e.keepLeaderAlive(ctx); err != nil {
				e.status.Store(uint32(StatusFollower))
				e.logger.Error("Error keeping leader alive", "error", err.Error())
			}
			break
		}

		time.Sleep(e.lockTTL)
	}
}

func (e *Elector) tryBecomeLeader(ctx context.Context) bool {
	result, err := e.redis.SetNX(ctx, leaderKey, e.id, e.lockTTL).Result()
	if err != nil {
		e.logger.Error("Error trying to become leader", "error", err.Error())
		return false
	}

	return result
}

func (e *Elector) keepLeaderAlive(ctx context.Context) error {
	ticker := time.NewTicker(e.lockTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := e.redis.Expire(ctx, leaderKey, e.lockTTL).Result(); err != nil {
				return fmt.Errorf("error renewing leadership: %w", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}
