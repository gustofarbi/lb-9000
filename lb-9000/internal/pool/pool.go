package pool

import (
	"context"
	"fmt"
	"lb-9000/lb-9000/internal/orchestration"
	"lb-9000/lb-9000/internal/store"
	"lb-9000/lb-9000/internal/strategy"
	"log/slog"
	"net/http"
	"time"
)

type Pool struct {
	backendStore  store.Store
	strategy      strategy.Strategy
	orchestration orchestration.Orchestration

	logger *slog.Logger

	refreshRate time.Duration
	initialized bool
}

func New(
	store store.Store,
	strategy strategy.Strategy,
	orchestration orchestration.Orchestration,
	logger *slog.Logger,
	refreshRate time.Duration,
) *Pool {
	return &Pool{
		backendStore:  store,
		strategy:      strategy,
		orchestration: orchestration,
		logger:        logger,
		refreshRate:   refreshRate,
	}
}

func (p *Pool) Director(request *http.Request) {
	if !p.initialized {
		panic("pool not initialized")
	}

	ctx := request.Context()

	elected, err := p.strategy.Elect(ctx, p.backendStore)
	if err != nil {
		panic("electing a backend: " + err.Error())
	}
	if elected == nil {
		panic("no pods available")
	}

	minUrl := elected.URL()

	if err = p.backendStore.AddRequests(ctx, minUrl, 1); err != nil {
		p.logger.Error("error adding request to backend", "error", err)
		return
	}

	p.logger.Info(
		"request directed to pod",
		"podUrl", minUrl,
		"requests", elected.Count(),
	)

	if minUrl == "" {
		p.logger.Error("error getting url from backend")
		return
	}

	p.orchestration.DirectRequest(request, elected)
}

func (p *Pool) ModifyResponse(response *http.Response) error {
	id, err := p.orchestration.GetBackendIDFromResponse(response)
	if err != nil {
		p.logger.Error("error getting id from response", "error", err)
		return nil
	}

	if err = p.backendStore.AddRequests(context.Background(), id, -1); err != nil {
		p.logger.Error("error removing request from backend", "error", err)
		return nil
	}

	return nil
}

func (p *Pool) Init() {
	if p.initialized {
		return
	}

	go p.orchestration.StartObserver(p.backendStore)
	go p.startLogger()

	p.initialized = true
	if p.logger != nil {
		p.logger.Info("refreshing pods")
	}
}

func (p *Pool) startLogger() {
	for range time.Tick(p.refreshRate) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, p.refreshRate)
		iterator, err := p.backendStore.Iterate(ctx)
		if err != nil {
			p.logger.Error("cannot iterate backends", "error", err)
			cancel()
			return
		}

		cancel()

		for backend := range iterator {
			p.logger.Info(fmt.Sprintf("pod '%s' has '%d' requests", backend.URL(), backend.Count()))

		}
	}
}
