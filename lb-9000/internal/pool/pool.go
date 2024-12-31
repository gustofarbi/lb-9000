package pool

import (
	"errors"
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

	elected := p.strategy.Elect(p.backendStore)
	if elected == nil {
		panic("no pods available")
	}

	minUrl := elected.URL()

	p.backendStore.AddRequests(minUrl, 1)

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
	name, err := p.orchestration.GetBackendNameFromResponse(response)
	if err != nil {
		p.logger.Error("error getting name from response", "error", err)
		return nil
	}

	p.backendStore.AddRequests(name, -1)

	return nil
}

func (p *Pool) Init() error {
	if p.initialized {
		return errors.New("pool already initialized")
	}

	go p.orchestration.StartObserver(p.backendStore)
	go p.startLogger()

	p.initialized = true
	if p.logger != nil {
		p.logger.Info("refreshing pods")
	}

	return nil
}

func (p *Pool) startLogger() {
	for range time.Tick(p.refreshRate) {
		p.backendStore.DebugPrint()
	}
}
