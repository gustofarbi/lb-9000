package main

import (
	"context"
	"fmt"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	appconfig "lb-9000/lb-9000/internal/config"
	"lb-9000/lb-9000/internal/orchestration"
	"lb-9000/lb-9000/internal/pool"
	"lb-9000/lb-9000/internal/proxy"
	"lb-9000/lb-9000/internal/store/memory"
	"lb-9000/lb-9000/internal/strategy"
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

	logger := slog.Default()

	watcher, err := clientset.
		CoreV1().
		Pods(appConfig.Specs.Namespace).
		Watch(ctx, metav1.ListOptions{
			LabelSelector: appConfig.Specs.Selector,
			FieldSelector: "status.phase=" + string(core.PodRunning),
		})

	if err != nil {
		return fmt.Errorf("error watching pods: %w", err)
	}

	podPool := pool.New(
		memory.NewMemoryStore(logger),
		strategy.FillHoles(),
		orchestration.NewKubernetes(logger, watcher, appConfig),
		logger,
		appConfig.RefreshRate,
	)

	if err = podPool.Init(); err != nil {
		return fmt.Errorf("initiating pool: %w", err)
	}

	proxy.Start(podPool, strconv.Itoa(appConfig.Specs.ContainerPort))

	return nil
}
