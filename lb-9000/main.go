package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	appconfig "lb-9000/internal/config"
	"lb-9000/internal/pool"
	"lb-9000/internal/proxy"
	"log/slog"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

func run() error {
	appConfig, err := appconfig.Parse("lb-9000/internal/config/config.yaml")
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	clientset, err := func() (*kubernetes.Clientset, error) {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(config)
	}()

	if err != nil {
		return fmt.Errorf("creating clientset: %w", err)
	}

	ctx := context.Background()

	podPool := pool.New(clientset, appConfig.Specs, slog.Default())
	err = podPool.Init(ctx)

	if err != nil {
		return fmt.Errorf("initiating pool: %w", err)
	}

	proxy.Start(podPool, "8080")

	return nil
}
