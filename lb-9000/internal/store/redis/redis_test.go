package redis

import (
	"context"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"lb-9000/lb-9000/internal/backend"
	"lb-9000/lb-9000/internal/config"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedis(t *testing.T) {
	request := testcontainers.ContainerRequest{
		Image:        "redis:7.4.1-bookworm",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	ctx := context.Background()

	redisC, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: request,
			Started:          true,
		},
	)

	assert.NoError(t, err)

	defer func() {
		if err = redisC.Terminate(ctx); err != nil {
			assert.NoError(t, err)
		}
	}()

	endpoint, err := redisC.Endpoint(ctx, "")

	cfg, err := config.Parse("../../config/.env")
	assert.NoError(t, err)

	cfg.StoreAddr = endpoint

	store := New(slog.Default(), cfg)

	instance := backend.NewBackend("http://localhost:8080", "test")

	err = store.Add(ctx, instance)
	assert.NoError(t, err)

	err = store.AddRequests(ctx, instance.URL(), 1)
	assert.NoError(t, err)

	iterator, err := store.Iterate(ctx)
	assert.NoError(t, err)

	i := 0

	for backendInstance := range iterator {
		i++
		assert.Equal(t, "http://localhost:8080", backendInstance.URL())
		assert.Equal(t, int64(1), backendInstance.Count())
		assert.Equal(t, "test", backendInstance.Name())
	}

	assert.Equal(t, 1, i)

	err = store.Remove(ctx, instance.URL())
	assert.NoError(t, err)

	iterator, err = store.Iterate(ctx)
	assert.NoError(t, err)

	i = 0

	for range iterator {
		i++
	}

	assert.Equal(t, 0, i)
}
