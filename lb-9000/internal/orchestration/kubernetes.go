package orchestration

import (
	"context"
	"fmt"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"lb-9000/lb-9000/internal/backend"
	"lb-9000/lb-9000/internal/config"
	"lb-9000/lb-9000/internal/store"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type kubernetes struct {
	logger  *slog.Logger
	watcher watch.Interface
	config  *config.Config
}

func (k *kubernetes) DirectRequest(
	request *http.Request,
	backend *backend.Backend,
) {
	request.URL.Scheme = "http"
	request.URL.Host = fmt.Sprintf(
		"%s.%s.%s.svc.cluster.local:%d",
		strings.Replace(backend.URL(), ".", "-", -1),
		k.config.Specs.ServiceName,
		k.config.Specs.Namespace,
		k.config.Specs.ContainerPort,
	)
}

func (k *kubernetes) GetBackendNameFromResponse(response *http.Response) (string, error) {
	ip, err := getIpFromHost(response.Request.URL.Host)
	if err != nil {
		return "", fmt.Errorf("error getting ip from host: %w", err)
	}

	return ip, nil
}

func (k *kubernetes) StartObserver(store store.Store) {
	ctx := context.Background()

	for event := range k.watcher.ResultChan() {
		podFromEvent, ok := event.Object.(*core.Pod)
		if !ok {
			if k.logger != nil {
				k.logger.Error("unexpected object type", "object", event.Object)
			}
			continue
		}

		switch event.Type {
		case watch.Added:
			// when a pod is added, it needs to be added to the pool
			// at this time the pod may not have a URL assigned yet
			k.addBackend(ctx, store, podFromEvent)
		case watch.Deleted:
			// when a pod is deleted, it needs to be removed from the pool
			k.removeBackend(ctx, store, podFromEvent)
		case watch.Modified:
			// there are several cases when a pod is modified:
			// 1. the pod is being deleted -> it will have a deletion timestamp
			// 2. the pod changed state and is now running -> it will have an URL
			if podFromEvent.DeletionTimestamp != nil {
				k.removeBackend(ctx, store, podFromEvent)
			} else if podFromEvent.Status.PodIP != "" {
				// todo look at the state here maybe?
				k.addBackend(ctx, store, podFromEvent)
			}
		}
	}
}

func (k *kubernetes) removeBackend(
	ctx context.Context,
	store store.Store,
	pod *core.Pod,
) {
	if err := store.Remove(ctx, pod.Status.PodIP); err != nil {
		if k.logger != nil {
			k.logger.Error("error removing backend", "error", err)
		}
	}
}

func (k *kubernetes) addBackend(
	ctx context.Context,
	store store.Store,
	pod *core.Pod,
) {
	if err := store.Add(ctx, backend.NewBackend(pod.Status.PodIP, pod.Name)); err != nil {
		if k.logger != nil {
			k.logger.Error("error adding backend", "error", err)
		}
	}
}

func getIpFromHost(host string) (string, error) {
	if parsedUrl, err := url.ParseRequestURI(host); err == nil {
		host = parsedUrl.Host
	}

	ip, _, ok := strings.Cut(host, ".")
	if !ok {
		return "", fmt.Errorf("expected to be able to cut host")
	}

	ip = strings.Replace(ip, "-", ".", -1)
	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return "", fmt.Errorf("could not parse ip '%s'", ip)
	}

	return ip, nil
}
