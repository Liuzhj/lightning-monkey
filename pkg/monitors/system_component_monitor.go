package monitors

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubernetesSystemComponentMonitor struct {
	clusterId string
	client    *k8s.KubernetesClientSet
	cache     *[]entities.WatchPoint
	stopChan  chan int
}

func (m *KubernetesSystemComponentMonitor) GetName() string {
	return "Kubernetes System Component Monitor"
}

func (m *KubernetesSystemComponentMonitor) GetWatchPoints() []entities.WatchPoint {
	return *m.cache
}

func (m *KubernetesSystemComponentMonitor) Start() error {
	if m.stopChan == nil {
		m.stopChan = make(chan int)
	}
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("Unhandled exception from \"%s\", error: %s", m.GetName(), err.(error).Error())
			}
		}()
		for {
			if m.stopChan == nil {
				return
			}
			select {
			case _, isOpen := <-m.stopChan:
				if !isOpen {
					return
				}
			default:
				m.doMonitor()
			}
			time.Sleep(time.Second * 10)
		}
	}()
	return nil
}

func (m *KubernetesSystemComponentMonitor) doMonitor() {
	if m.client == nil {
		return
	}
	csl, err := m.client.CoreClient.CoreV1().ComponentStatuses().List(meta_v1.ListOptions{})
	if err != nil {
		logrus.Errorf("Failed to list system components from cluster: %s, error: %s", m.clusterId, err.Error())
		return
	}
	if csl == nil || csl.Items == nil || len(csl.Items) == 0 {
		return
	}
	var item v1.ComponentStatus
	wps := make([]entities.WatchPoint, 0, len(csl.Items))
	for i := 0; i < len(csl.Items); i++ {
		item = csl.Items[i]
		wp := entities.WatchPoint{}
		wp.Name = item.Name
		wp.Namespace = "-"
		wp.Status = getHealthStatus(item.Conditions)
		wp.LastCheckTime = time.Now()
		wp.IsSystemComponent = true
		wps = append(wps, wp)
	}
	atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&m.cache)), unsafe.Pointer(&wps))
}

func getHealthStatus(cc []v1.ComponentCondition) string {
	if cc == nil || len(cc) == 0 {
		return Unknown
	}
	for i := 0; i < len(cc); i++ {
		if cc[i].Type == v1.ComponentHealthy {
			if cc[i].Status == v1.ConditionTrue {
				return Healthy
			} else if cc[i].Status == v1.ConditionUnknown {
				return Unknown
			} else {
				return Unhealthy
			}
		}
	}
	return Unhealthy
}

func (m *KubernetesSystemComponentMonitor) Dispose() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}
}
