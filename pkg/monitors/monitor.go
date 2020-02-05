package monitors

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"strings"
)

const (
	Healthy   string = "healthy"
	Unhealthy string = "unhealthy"
	Unknown   string = "unknown"
)

type KubernetesResourceMonitor interface {
	GetName() string
	GetWatchPoints() []entities.WatchPoint
	Start() error
	Dispose()
}

func NewMonitor(t string, c *k8s.KubernetesClientSet, clusterId string) KubernetesResourceMonitor {
	switch strings.ToLower(t) {
	case "sys":
		return &KubernetesSystemComponentMonitor{
			clusterId: clusterId,
			client:    c,
		}
	case "deployment":
		return &KubernetesDeploymentMonitor{
			clusterId: clusterId,
			client:    c,
		}
	case "daemonset":
		return &KubernetesDaemonSetMonitor{
			clusterId: clusterId,
			client:    c,
		}
	default:
		return nil
	}
}
