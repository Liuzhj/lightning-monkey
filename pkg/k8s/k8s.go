package k8s

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8s "k8s.io/client-go/kubernetes"
	ko "k8s.io/client-go/pkg/api/v1"
	ko_v1beta "k8s.io/client-go/pkg/apis/apps/v1beta1"
	ko_v2alpha1 "k8s.io/client-go/pkg/apis/batch/v2alpha1"
	ko_ext_v1beta "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	v1 "k8s.io/client-go/pkg/apis/storage/v1"
	"os/exec"
	"path/filepath"
)

const (
	kubeletSettings string = `address: 0.0.0.0
apiVersion: kubelet.config.k8s.io/v1beta1
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
cgroupDriver: cgroupfs
clusterDNS:
- 10.96.0.10
clusterDomain: cluster.local
configMapAndSecretChangeDetectionStrategy: Watch
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
contentType: application/vnd.kubernetes.protobuf
cpuCFSQuota: true
cpuCFSQuotaPeriod: 100ms
cpuManagerPolicy: none
cpuManagerReconcilePeriod: 10s
enableControllerAttachDetach: true
enableDebuggingHandlers: true
eventBurst: 10
eventRecordQPS: 5
evictionHard:
  imagefs.available: 15%
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 127.0.0.1
healthzPort: 10248
httpCheckFrequency: 20s
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
iptablesDropBit: 15
iptablesMasqueradeBit: 14
kind: KubeletConfiguration
kubeAPIBurst: 10
kubeAPIQPS: 5
makeIPTablesUtilChains: true
maxOpenFiles: 1000000
maxPods: 110
nodeLeaseDurationSeconds: 40
nodeStatusReportFrequency: 1m0s
nodeStatusUpdateFrequency: 10s
oomScoreAdj: -999
podPidsLimit: -1
port: 10250
registryBurst: 10
registryPullQPS: 5
resolvConf: /etc/resolv.conf
rotateCertificates: true
runtimeRequestTimeout: 2m0s
serializeImagePulls: false
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s`
)

func GenerateKubeletConfig(certPath, masterAPIAddr string) error {
	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("kubeadm init phase kubeconfig kubelet --cert-dir=%s --kubeconfig-dir=%s --apiserver-advertise-address=%s", certPath, certPath, masterAPIAddr))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	reader := bufio.NewReader(stdout)
	for {
		traceData, _, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		logrus.Infof(string(traceData))
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(certPath, "kubelet_settings.yml"), []byte(kubeletSettings), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write kubelet settings file, error: %s", err.Error())
	}
	return nil
}

func CreateK8SResource(client *k8s.Clientset, obj runtime.Object) (runtime.Object, error) {
	var o runtime.Object
	var err error
	metadata, _ := meta_v1.ObjectMetaFor(obj)
	switch obj.(type) {
	case *ko.ReplicationController:
		o, err = client.ReplicationControllers(metadata.Namespace).Create(obj.(*ko.ReplicationController))
	case *ko.Service:
		o, err = client.Services(metadata.Namespace).Create(obj.(*ko.Service))
	case *ko.Pod:
		o, err = client.Pods(metadata.Namespace).Create(obj.(*ko.Pod))
	case *ko_v1beta.Deployment:
		o, err = client.AppsV1beta1().Deployments(metadata.Namespace).Create(obj.(*ko_v1beta.Deployment))
	case *ko_v2alpha1.CronJob:
		o, err = client.CronJobs(metadata.Namespace).Create(obj.(*ko_v2alpha1.CronJob))
	case *ko_ext_v1beta.DaemonSet:
		o, err = client.DaemonSets(metadata.Namespace).Create(obj.(*ko_ext_v1beta.DaemonSet))
	case *ko.ConfigMap:
		o, err = client.ConfigMaps(metadata.Namespace).Create(obj.(*ko.ConfigMap))
	case *ko.ServiceAccount:
		o, err = client.ServiceAccounts(metadata.Namespace).Create(obj.(*ko.ServiceAccount))
	case *ko_ext_v1beta.Ingress:
		o, err = client.Ingresses(metadata.Namespace).Create(obj.(*ko_ext_v1beta.Ingress))
	case *ko.PersistentVolumeClaim:
		o, err = client.PersistentVolumeClaims(metadata.Namespace).Create(obj.(*ko.PersistentVolumeClaim))
	case *ko.PersistentVolume:
		o, err = client.PersistentVolumes().Create(obj.(*ko.PersistentVolume))
	default:
		return nil, fmt.Errorf("Unsupported obj kind %s", obj.GetObjectKind().GroupVersionKind().Kind)
	}
	if err != nil {
		return nil, err
	}
	o.GetObjectKind().SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	return o, nil
}

func IsKubernetesResourceExists(client *k8s.Clientset, obj runtime.Object) (bool, error) {
	metadata, _ := meta_v1.ObjectMetaFor(obj)
	switch obj.(type) {
	case *ko.ReplicationController:
		realObj, err := client.ReplicationControllers(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.Service:
		realObj, err := client.Services(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.Pod:
		realObj, err := client.Pods(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_v1beta.Deployment:
		realObj, err := client.AppsV1beta1().Deployments(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_v2alpha1.CronJob:
		realObj, err := client.CronJobs(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_ext_v1beta.DaemonSet:
		realObj, err := client.DaemonSets(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.ConfigMap:
		realObj, err := client.ConfigMaps(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.ServiceAccount:
		realObj, err := client.ServiceAccounts(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_ext_v1beta.Ingress:
		realObj, err := client.Ingresses(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.PersistentVolumeClaim:
		realObj, err := client.PersistentVolumeClaims(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.PersistentVolume:
		realObj, err := client.PersistentVolumes().Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	default:
	}
	if sc, ok := obj.(*v1.StorageClass); ok {
		realObj, err := client.StorageV1Client.StorageClasses().Get(sc.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	}
	return false, errors.New("Unsupported obj kind: " + obj.GetObjectKind().GroupVersionKind().Kind)
}
