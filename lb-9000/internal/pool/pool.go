package pool

import (
	"context"
	"errors"
	"fmt"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"lb-9000/lb-9000/internal/config"
	"lb-9000/lb-9000/internal/pod"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Pool struct {
	podMap *pod.Map

	clientset *kubernetes.Clientset
	logger    *slog.Logger
	watcher   watch.Interface

	initialized bool

	cfg *config.Config
}

func New(
	clientset *kubernetes.Clientset,
	cfg *config.Config,
	logger *slog.Logger,
) *Pool {
	return &Pool{
		podMap:    pod.NewPodMap(logger),
		clientset: clientset,
		cfg:       cfg,
		logger:    logger,
	}
}

func (p *Pool) Director(request *http.Request) {
	if !p.initialized {
		panic("pool not initialized")
	}

	elected := p.podMap.Elect()
	minIp := elected.IP()

	p.podMap.Delta(minIp, 1)

	p.logger.Info(
		"request directed to pod",
		"podIp", minIp,
		"requests", elected.Count(),
	)

	if minIp == "" {
		// todo what to do here?
		return
	}

	request.URL.Scheme = "http"
	request.URL.Host = fmt.Sprintf(
		"%s.%s.%s.svc.cluster.local:%d",
		strings.Replace(minIp, ".", "-", -1),
		p.cfg.Specs.ServiceName,
		p.cfg.Specs.Namespace,
		p.cfg.Specs.ContainerPort,
	)
}

func (p *Pool) ModifyResponse(response *http.Response) error {
	ip, err := getIpFromHost(response.Request.URL.Host)
	if err != nil {
		return fmt.Errorf("error getting ip from host: %w", err)
	}

	// one less request to this pod
	p.podMap.Delta(ip, -1)

	return nil
}

func (p *Pool) Init(ctx context.Context) error {
	if p.initialized {
		return errors.New("pool already initialized")
	}

	watcher, err := p.clientset.
		CoreV1().
		Pods(p.cfg.Specs.Namespace).
		Watch(ctx, metav1.ListOptions{
			LabelSelector: p.cfg.Specs.Selector,
			FieldSelector: "status.phase=" + string(core.PodRunning),
		})

	if err != nil {
		return fmt.Errorf("error watching pods: %w", err)
	}

	p.watcher = watcher

	go p.refreshLoop()
	go p.startLogger()

	p.initialized = true
	p.logger.Info("refreshing pods")

	return nil
}

func (p *Pool) Stop() {
	p.watcher.Stop()
}

func (p *Pool) refreshLoop() {
	for event := range p.watcher.ResultChan() {
		podFromEvent, ok := event.Object.(*core.Pod)
		if !ok {
			p.logger.Error("unexpected object type", "object", event.Object)
			continue
		}

		switch event.Type {
		case watch.Added:
			// when a pod is added, it needs to be added to the pool
			// at this time the pod may not have an IP assigned yet
			p.podMap.Add(podFromEvent.Status.PodIP, podFromEvent.Name)
		case watch.Deleted:
			// when a pod is deleted, it needs to be removed from the pool
			p.podMap.Delete(podFromEvent.Status.PodIP)
		case watch.Modified:
			// there are several cases when a pod is modified:
			// 1. the pod is being deleted -> it will have a deletion timestamp
			// 2. the pod changed state and is now running -> it will have an IP
			if podFromEvent.DeletionTimestamp != nil {
				p.podMap.Delete(podFromEvent.Status.PodIP)
			} else if podFromEvent.Status.PodIP != "" {
				// todo look at the state here maybe?
				p.podMap.Add(podFromEvent.Status.PodIP, podFromEvent.Name)
			}
		}
	}
}

func (p *Pool) startLogger() {
	for range time.Tick(p.cfg.RefreshRate) {
		p.podMap.DebugPrint()
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
