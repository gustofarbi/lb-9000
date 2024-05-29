package pool

import (
	"context"
	"errors"
	"fmt"
	"lb-9000/internal/config"
	"lb-9000/internal/pod"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type Pool struct {
	podMap *pod.PodMap

	clientset *kubernetes.Clientset
	logger    *slog.Logger

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

	pod := p.podMap.Elect()
	minIp := pod.IP()

	p.podMap.Delta(minIp, 1)

	p.logger.Info(fmt.Sprintf("request directed to '%s' which has '%d' requests", minIp, pod.Count()))

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
		p.logger.Error("error getting ip from host", "err", err)
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

	go p.refreshLoop(watcher)
	go p.startLogger()

	p.initialized = true
	p.logger.Info("refreshing pods")

	return nil
}

func (p *Pool) refreshLoop(watcher watch.Interface) {
	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*core.Pod)
		if !ok {
			p.logger.Error("unexpected object type", "object", event.Object)
			continue
		}

		switch event.Type {
		case watch.Added:
			// when a pod is added, it needs to be added to the pool
			// at this time the pod may not have an IP assigned yet
			p.podMap.Add(pod.Status.PodIP, pod.Name)
		case watch.Deleted:
			// when a pod is deleted, it needs to be removed from the pool
			p.podMap.Delete(pod.Status.PodIP)
		case watch.Modified:
			// there are several cases when a pod is modified:
			// 1. the pod is being deleted -> it will have a deletion timestamp
			// 2. the pod changed state and is now running -> it will have an IP
			if pod.DeletionTimestamp != nil {
				p.podMap.Delete(pod.Status.PodIP)
			} else if pod.Status.PodIP != "" {
				// todo look at the state here maybe?
				p.podMap.Add(pod.Status.PodIP, pod.Name)
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
	// todo idk this should be done in a different way
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}

	parsedUrl, err := url.ParseRequestURI(host)
	if err != nil {
		return "", fmt.Errorf("could not parse url '%s': %w", host, err)
	}

	ip, _, ok := strings.Cut(parsedUrl.Host, ".")
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
