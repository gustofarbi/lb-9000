package main

import (
	"context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	appconfig "lb-9000/internal/config"
	"lb-9000/internal/pool"
	"lb-9000/internal/proxy"
	"log/slog"
)

func main() {
	appConfig, err := appconfig.Parse("lb-9000/internal/config/config.yaml")
	if err != nil {
		panic(err)
	}

	var clientset *kubernetes.Clientset

	{
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
		clientset = kubernetes.NewForConfigOrDie(config)
	}

	ctx := context.Background()
	podPool := pool.New(clientset, appConfig.Specs, slog.Default())
	err = podPool.Init(ctx)
	if err != nil {
		panic(err)
	}

	proxy.Start(podPool, "8080")
}
