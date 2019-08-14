package monitors

import "time"

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
	Status            string    `json:"status"`
	LastCheckTime     time.Time `json:"last_check_time"`
}
