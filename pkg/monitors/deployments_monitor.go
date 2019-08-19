package monitors

import (
	"github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync/atomic"
	"time"
	"unsafe"
)

type KubernetesDeploymentMonitor struct {
	clusterId string
	client    *kubernetes.Clientset
	cache     *[]WatchPoint
	stopChan  chan int
}

func (m *KubernetesDeploymentMonitor) GetName() string {
	return "Kubernetes Deployment Monitor"
}

func (m *KubernetesDeploymentMonitor) GetWatchPoints() []WatchPoint {
	return *m.cache
}

func (m *KubernetesDeploymentMonitor) Start() error {
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

func (m *KubernetesDeploymentMonitor) Dispose() {
	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}
}

func (m *KubernetesDeploymentMonitor) doMonitor() {
	if m.client == nil {
		return
	}
	wps := m.getAppsV1DeploymentsStatus("kube-system")
	//wps2 := m.getAppsV1BetaDeploymentsStatus("kube-system")
	//if wps2 != nil && len(wps2) > 0 {
	//	wps = append(wps, wps2...)
	//}
	atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&m.cache)), unsafe.Pointer(&wps))
}

func (m *KubernetesDeploymentMonitor) getAppsV1DeploymentsStatus(namespace string) []WatchPoint {
	csl, err := m.client.AppsV1().Deployments(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		logrus.Errorf("Failed to list deployments from cluster: %s, error: %s", m.clusterId, err.Error())
		return nil
	}
	if csl == nil || csl.Items == nil || len(csl.Items) == 0 {
		return nil
	}
	var item apps_v1.Deployment
	wps := make([]WatchPoint, 0, len(csl.Items))
	for i := 0; i < len(csl.Items); i++ {
		item = csl.Items[i]
		wp := WatchPoint{}
		wp.Name = item.Name
		wp.Namespace = namespace
		wp.Status = getDeploymentHealthStatus(item.Status.Conditions)
		wp.LastCheckTime = time.Now()
		wp.IsSystemComponent = false
		wps = append(wps, wp)
	}
	return wps
}

func getDeploymentHealthStatus(cc []apps_v1.DeploymentCondition) string {
	if cc == nil || len(cc) == 0 {
		return Unknown
	}
	for i := 0; i < len(cc); i++ {
		if cc[i].Type == apps_v1.DeploymentAvailable {
			if cc[i].Status != v1.ConditionTrue {
				return Unhealthy
			}
		} else if cc[i].Type == apps_v1.DeploymentProgressing {
			if cc[i].Status != v1.ConditionTrue {
				return Unhealthy
			}
		} else {
			logrus.Errorf("Unknown condition type in Kubernetes deployment monitor: %s", cc[i].Type)
			return Unhealthy
		}
	}
	return Healthy
}

//func (m *KubernetesDeploymentMonitor) getAppsV1BetaDeploymentsStatus(namespace string) []WatchPoint {
//	csl, err := m.client.AppsV1beta1().Deployments(namespace).List(meta_v1.ListOptions{})
//	if err != nil {
//		logrus.Errorf("Failed to list deployments from cluster: %s, error: %s", m.clusterId, err.Error())
//		return nil
//	}
//	if csl == nil || csl.Items == nil || len(csl.Items) == 0 {
//		return nil
//	}
//	var item ko_v1beta.Deployment
//	wps := make([]WatchPoint, 0, len(csl.Items))
//	for i := 0; i < len(csl.Items); i++ {
//		item = csl.Items[i]
//		wp := WatchPoint{}
//		wp.Name = item.Name
//		wp.Namespace = namespace
//		wp.Status = getV1BetaDeploymentHealthStatus(item.Status.Conditions)
//		wp.LastCheckTime = time.Now()
//		wp.IsSystemComponent = false
//		wps = append(wps, wp)
//	}
//	return wps
//}

//func getV1BetaDeploymentHealthStatus(cc []ko_v1beta.DeploymentCondition) string {
//	if cc == nil || len(cc) == 0 {
//		return Unknown
//	}
//	for i := 0; i < len(cc); i++ {
//		if cc[i].Type == ko_v1beta.DeploymentAvailable {
//			if cc[i].Status != v1.ConditionTrue {
//				return Unhealthy
//			}
//		} else if cc[i].Type == ko_v1beta.DeploymentProgressing {
//			if cc[i].Status != v1.ConditionTrue {
//				return Unhealthy
//			}
//		} else {
//			logrus.Errorf("Unknown condition type in Kubernetes deployment monitor: %s", cc[i].Type)
//			return Unhealthy
//		}
//	}
//	return Healthy
//}
