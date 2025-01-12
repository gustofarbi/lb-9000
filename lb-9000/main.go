package main

import (
	"fmt"
	appconfig "lb-9000/lb-9000/internal/config"
	"lb-9000/lb-9000/internal/election"
	"lb-9000/lb-9000/internal/orchestration"
	"lb-9000/lb-9000/internal/pool"
	"lb-9000/lb-9000/internal/proxy"
	"lb-9000/lb-9000/internal/store"
	"lb-9000/lb-9000/internal/strategy"
	"lb-9000/lb-9000/internal/utils"
	"log/slog"
	"os"
	"strconv"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

func run() error {
	appConfig, err := appconfig.Parse("lb-9000/internal/config/.env")
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	logger := slog.Default()

	orchestrator, err := orchestration.NewKubernetes(logger, appConfig)
	if err != nil {
		return fmt.Errorf("creating orchestrator: %w", err)
	}

	elector := election.NewElector(
		orchestrator.InstanceID(),
		logger,
		utils.GetRedisClient(appConfig),
		appConfig.LockTTL,
	)

	podPool := pool.New(
		store.Get(appConfig, logger),
		strategy.FillHoles(),
		orchestrator,
		elector,
		logger,
		appConfig.RefreshRate,
	)

	proxy.Start(podPool, strconv.Itoa(appConfig.ContainerPort))

	return nil
}
