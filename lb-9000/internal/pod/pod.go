package pod

import (
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
)

type Pod struct {
	ip    string
	name  string
	count *atomic.Int64
}

func (p *Pod) IP() string {
	return p.ip
}

func (p *Pod) Count() int64 {
	return p.count.Load()
}

type PodMap struct {
	inner  map[string]*Pod
	logger *slog.Logger
}

func NewPodMap(logger *slog.Logger) *PodMap {
	if logger == nil {
		logger = slog.Default()
	}

	return &PodMap{
		inner:  make(map[string]*Pod),
		logger: logger,
	}
}

func NewPod(ip, name string) *Pod {
	return &Pod{
		ip:    ip,
		name:  name,
		count: new(atomic.Int64),
	}
}

func (p *PodMap) Add(ip, name string) {
	p.logger.Info("adding", "ip", ip, "name", name)

	p.inner[ip] = NewPod(ip, name)
}

func (p *PodMap) Delete(ip string) {
	if ip == "" {
		return
	}

	p.logger.Info(fmt.Sprintf("pod '%s' deleted", ip))
	delete(p.inner, ip)
}

func (p *PodMap) Elect() *Pod {
	var (
		minCount int64 = math.MaxInt64
		minPod   *Pod
	)

	// make this thread safe
	// todo naive implementation: is it fast enough?
	for _, pod := range p.inner {
		if count := pod.count.Load(); count < minCount {
			minCount = count
			minPod = pod
		}
		if minCount == 0 {
			break
		}
	}

	return minPod
}

func (p *PodMap) Delta(ip string, delta int64) {
	if ip == "" {
		return
	}

	pod, ok := p.inner[ip]
	if !ok {
		p.logger.Info("could not find pod", "ip", ip)
		return
	}

	newCount := max(0, pod.count.Load()+delta)
	pod.count.Store(newCount)
}

func (p *PodMap) DebugPrint() {
	for _, pod := range p.inner {
		p.logger.Info(fmt.Sprintf("pod '%s' has '%d' requests", pod.ip, pod.count.Load()))
	}
}
