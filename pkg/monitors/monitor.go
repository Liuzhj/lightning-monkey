package monitors

import (
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

const (
	Healthy   string = "healthy"
	Unhealthy string = "unhealthy"
	Unknown   string = "unknown"
)

type KubernetesResourceMonitor interface {
	GetName() string
	GetWatchPoints() []WatchPoint
	Start() error
	Dispose()
}

type WatchPoint struct {
	IsSystemComponent bool      `json:"is_system_component"`
	Name              string    `json:"name"`
	Namespace         string    `json:"namespace"`
	Status            string    `json:"status"`
	LastCheckTime     time.Time `json:"last_check_time"`
}

func NewMonitor(t string, c *kubernetes.Clientset, clusterId string) KubernetesResourceMonitor {
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
