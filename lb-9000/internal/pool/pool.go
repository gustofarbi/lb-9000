package pool

import (
	"context"
	"errors"
	"fmt"
	cmap "github.com/orcaman/concurrent-map/v2"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"lb-9000/internal/config"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Pool struct {
	sync.Mutex
	pods cmap.ConcurrentMap[string, int]

	clientset *kubernetes.Clientset
	logger    *slog.Logger

	initialized bool

	specs config.Specs
}

func New(
	clientset *kubernetes.Clientset,
	specs config.Specs,
	logger *slog.Logger,
) *Pool {
	return &Pool{
		pods:      cmap.New[int](),
		clientset: clientset,
		specs:     specs,
		logger:    logger,
	}
}

func (p *Pool) Director(request *http.Request) {
	if !p.initialized {
		panic("pool not initialized")
	}

	minCount := math.MaxInt
	var minIp string

	p.Lock()
	// todo naive implementation: is it fast enough?
	for ip := range p.pods.IterBuffered() {
		if ip.Val < minCount {
			minCount = ip.Val
			minIp = ip.Key
		}
		if minCount == 0 {
			break
		}
	}

	p.increaseCount(minIp, 1)
	p.Unlock()

	p.logger.Info(fmt.Sprintf("request directed to '%s' which has '%d' requests", minIp, minCount))

	if minIp == "" {
		// todo what to do here?
		return
	}

	request.URL.Scheme = "http"
	request.URL.Host = fmt.Sprintf(
		"%s.%s.%s.svc.cluster.local:%d",
		strings.Replace(minIp, ".", "-", -1),
		p.specs.ServiceName,
		p.specs.Namespace,
		p.specs.ContainerPort,
	)
}

func (p *Pool) ModifyResponse(response *http.Response) error {
	ip, err := getIpFromHost(response.Request.URL.Host)
	if err != nil {
		p.logger.Error("error getting ip from host", "err", err)
	}

	// one less request to this pod
	p.increaseCount(ip, -1)

	return nil
}

func (p *Pool) Init(ctx context.Context) error {
	if p.initialized {
		return errors.New("pool already initialized")
	}

	watcher, err := p.clientset.
		CoreV1().
		Pods(p.specs.Namespace).
		Watch(ctx, metav1.ListOptions{
			LabelSelector: p.specs.Selector,
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
			p.add(pod.Status.PodIP)
		case watch.Deleted:
			// when a pod is deleted, it needs to be removed from the pool
			p.remove(pod.Status.PodIP)
		case watch.Modified:
			// there are several cases when a pod is modified:
			// 1. the pod is being deleted -> it will have a deletion timestamp
			// 2. the pod changed state and is now running -> it will have an IP
			if pod.DeletionTimestamp != nil {
				p.remove(pod.Status.PodIP)
			} else if pod.Status.PodIP != "" {
				// todo look at the state here maybe?
				p.add(pod.Status.PodIP)
			}
		}
	}
}

func (p *Pool) add(ip string) {
	if ip == "" {
		return
	}

	p.logger.Info(fmt.Sprintf("pod '%s' added", ip))
	p.pods.Set(ip, 0)
}

func (p *Pool) remove(ip string) {
	if ip == "" {
		return
	}

	if _, ok := p.pods.Get(ip); !ok {
		return
	}

	p.logger.Info(fmt.Sprintf("pod '%s' deleted", ip))
	p.pods.Remove(ip)
}

func (p *Pool) increaseCount(ip string, delta int) {
	if ip == "" {
		return
	}

	count, ok := p.pods.Get(ip)
	if !ok {
		return
	}

	p.pods.Set(ip, count+delta)
}

func (p *Pool) startLogger() {
	for range time.Tick(1 * time.Second) {
		p.Lock()

		for pod := range p.pods.IterBuffered() {
			p.logger.Info(fmt.Sprintf("pod '%s' has '%d' requests", pod.Key, pod.Val))
		}
		p.Unlock()
	}
}
func getIpFromHost(host string) (string, error) {
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
