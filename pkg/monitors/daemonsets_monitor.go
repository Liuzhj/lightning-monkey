package monitors

import (
	"github.com/sirupsen/logrus"
	ko_ext_v1beta "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync/atomic"
	"time"
	"unsafe"
)

type KubernetesDaemonSetMonitor struct {
	clusterId string
	client    *kubernetes.Clientset
	cache     *[]WatchPoint
	stopChan  chan int
}

func (m *KubernetesDaemonSetMonitor) GetName() string {
	return "Kubernetes DaemonSet Monitor"
}

func (m *KubernetesDaemonSetMonitor) GetWatchPoints() []WatchPoint {
	return *m.cache
}

func (m *KubernetesDaemonSetMonitor) Start() error {
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

func (m *KubernetesDaemonSetMonitor) Dispose() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}
}

func (m *KubernetesDaemonSetMonitor) doMonitor() {
	if m.client == nil {
		return
	}
	wps := m.getDaemonSetsStatus("kube-system")
	atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&m.cache)), unsafe.Pointer(&wps))
}

func (m *KubernetesDaemonSetMonitor) getDaemonSetsStatus(namespace string) []WatchPoint {
	csl, err := m.client.ExtensionsV1beta1().DaemonSets(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		logrus.Errorf("Failed to list daemonsets from cluster: %s, error: %s", m.clusterId, err.Error())
		return nil
	}
	if csl == nil || csl.Items == nil || len(csl.Items) == 0 {
		return nil
	}
	var item ko_ext_v1beta.DaemonSet
	wps := make([]WatchPoint, 0, len(csl.Items))
	for i := 0; i < len(csl.Items); i++ {
		item = csl.Items[i]
		wp := WatchPoint{}
		wp.Name = item.Name
		wp.Namespace = namespace
		wp.Status = getDaemonSetHealthStatus(item.Status)
		wp.LastCheckTime = time.Now()
		wp.IsSystemComponent = false
		wps = append(wps, wp)
	}
	return wps
}

func getDaemonSetHealthStatus(ds ko_ext_v1beta.DaemonSetStatus) string {
	if ds.NumberMisscheduled+ds.NumberUnavailable > 0 {
		return Unhealthy
	}
	return Healthy
}
