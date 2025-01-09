package orchestration

import (
	"context"
	"fmt"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"lb-9000/lb-9000/internal/config"
	"log/slog"
)

func NewKubernetes(
	logger *slog.Logger,
	config *config.Config,
) (Orchestration, error) {
	clientSet, err := func() (*k8s.Clientset, error) {
		k8sConfig, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return k8s.NewForConfig(k8sConfig)
	}()

	if err != nil {
		return nil, fmt.Errorf("creating client set: %w", err)
	}

	watcher, err := clientSet.
		CoreV1().
		Pods(config.Namespace).
		Watch(context.Background(), metav1.ListOptions{
			LabelSelector: config.Selector,
			FieldSelector: "status.phase=" + string(core.PodRunning),
		})

	if err != nil {
		return nil, fmt.Errorf("error watching pods: %w", err)
	}

	return &kubernetes{
		logger:  logger,
		watcher: watcher,
		config:  config,
	}, nil
}
