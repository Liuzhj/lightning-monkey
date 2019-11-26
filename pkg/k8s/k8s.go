package k8s

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/utils"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	apps_v1 "k8s.io/api/apps/v1"
	ko_v1beta "k8s.io/api/apps/v1beta1"
	ko_v2alpha1 "k8s.io/api/batch/v2alpha1"
	ko "k8s.io/api/core/v1"
	ext_v1beta "k8s.io/api/extensions/v1beta1"
	ko_ext_v1beta "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	rbac_v1beta "k8s.io/api/rbac/v1beta1"
	v1 "k8s.io/api/storage/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	agg_v1betaObj "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	agg_v1beta "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1beta1"
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
- {{.DNSIP}}
clusterDomain: {{.DOMAIN}}
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
  memory.available: 200Mi
  nodefs.available: 10%
  nodefs.inodesFree: 10%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 0.0.0.0
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
maxPods: {{.MAXPODS}}
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

type KubernetesClientSet struct {
	CoreClient          *kubernetes.Clientset
	APIRegClientV1beta1 *agg_v1beta.ApiregistrationV1beta1Client
}

func GenerateKubeletConfig(certPath, masterAPIAddr string, replacementSlots map[string]string) error {
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
	tpl, err := utils.TemplateReplace(kubeletSettings, map[string]string{
		"MAXPODS": replacementSlots[entities.MasterSettings_MaxPodCountPerNode],
		"DOMAIN":  replacementSlots[entities.MasterSettings_ServiceDNSDomain],
		"DNSIP":   replacementSlots[entities.MasterSettings_ServiceDNSClusterIP],
	})
	if err != nil {
		return fmt.Errorf("Failed to replace Kubelet configuration template content, error: %s", err.Error())
	}
	err = ioutil.WriteFile(filepath.Join(certPath, "kubelet_settings.yml"), []byte(tpl), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write kubelet settings file, error: %s", err.Error())
	}
	return nil
}

func CreateK8SResource(cs *KubernetesClientSet, obj runtime.Object) (runtime.Object, error) {
	var err error
	var o runtime.Object
	metadata, _ := utils.ObjectMetaFor(obj)
	switch obj.(type) {
	case *ko.ReplicationController:
		o, err = cs.CoreClient.CoreV1().ReplicationControllers(metadata.Namespace).Create(obj.(*ko.ReplicationController))
	case *ko.Service:
		o, err = cs.CoreClient.CoreV1().Services(metadata.Namespace).Create(obj.(*ko.Service))
	case *ko.Pod:
		o, err = cs.CoreClient.CoreV1().Pods(metadata.Namespace).Create(obj.(*ko.Pod))
	case *ko.Secret:
		o, err = cs.CoreClient.CoreV1().Secrets(metadata.Namespace).Create(obj.(*ko.Secret))
	case *ext_v1beta.Deployment:
		o, err = cs.CoreClient.ExtensionsV1beta1().Deployments(metadata.Namespace).Create(obj.(*ext_v1beta.Deployment))
	case *ko_v1beta.Deployment:
		o, err = cs.CoreClient.AppsV1beta1().Deployments(metadata.Namespace).Create(obj.(*ko_v1beta.Deployment))
	case *apps_v1.Deployment:
		o, err = cs.CoreClient.AppsV1().Deployments(metadata.Namespace).Create(obj.(*apps_v1.Deployment))
	case *ko_v2alpha1.CronJob:
		o, err = cs.CoreClient.BatchV2alpha1().CronJobs(metadata.Namespace).Create(obj.(*ko_v2alpha1.CronJob))
	case *ko_ext_v1beta.DaemonSet:
		o, err = cs.CoreClient.ExtensionsV1beta1().DaemonSets(metadata.Namespace).Create(obj.(*ko_ext_v1beta.DaemonSet))
	case *ko.ConfigMap:
		o, err = cs.CoreClient.CoreV1().ConfigMaps(metadata.Namespace).Create(obj.(*ko.ConfigMap))
	case *ko.ServiceAccount:
		o, err = cs.CoreClient.CoreV1().ServiceAccounts(metadata.Namespace).Create(obj.(*ko.ServiceAccount))
	case *ko_ext_v1beta.Ingress:
		o, err = cs.CoreClient.ExtensionsV1beta1().Ingresses(metadata.Namespace).Create(obj.(*ko_ext_v1beta.Ingress))
	case *ko.PersistentVolumeClaim:
		o, err = cs.CoreClient.CoreV1().PersistentVolumeClaims(metadata.Namespace).Create(obj.(*ko.PersistentVolumeClaim))
	case *ko.PersistentVolume:
		o, err = cs.CoreClient.CoreV1().PersistentVolumes().Create(obj.(*ko.PersistentVolume))
	case *rbac_v1beta.ClusterRole:
		o, err = cs.CoreClient.RbacV1beta1().ClusterRoles().Create(obj.(*rbac_v1beta.ClusterRole))
	case *rbac_v1beta.ClusterRoleBinding:
		o, err = cs.CoreClient.RbacV1beta1().ClusterRoleBindings().Create(obj.(*rbac_v1beta.ClusterRoleBinding))
	case *rbac_v1.ClusterRole:
		o, err = cs.CoreClient.RbacV1().ClusterRoles().Create(obj.(*rbac_v1.ClusterRole))
	case *rbac_v1.ClusterRoleBinding:
		o, err = cs.CoreClient.RbacV1().ClusterRoleBindings().Create(obj.(*rbac_v1.ClusterRoleBinding))
	case *agg_v1betaObj.APIService:
		o, err = cs.APIRegClientV1beta1.APIServices().Create(obj.(*agg_v1betaObj.APIService))
	case *rbac_v1beta.RoleBinding:
		o, err = cs.CoreClient.RbacV1beta1().RoleBindings(metadata.Namespace).Create(obj.(*rbac_v1beta.RoleBinding))
	case *apps_v1.StatefulSet:
		o, err = cs.CoreClient.AppsV1().StatefulSets(metadata.Namespace).Create(obj.(*apps_v1.StatefulSet))
	default:
		return nil, fmt.Errorf("Unsupported runtime.object kind %s", obj.GetObjectKind().GroupVersionKind().Kind)
	}
	if err != nil {
		return nil, err
	}
	return o, nil
}

func IsKubernetesResourceExists(cs *KubernetesClientSet, obj runtime.Object) (bool, error) {
	metadata, _ := utils.ObjectMetaFor(obj)
	switch obj.(type) {
	case *ko.ReplicationController:
		realObj, err := cs.CoreClient.CoreV1().ReplicationControllers(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.Service:
		realObj, err := cs.CoreClient.CoreV1().Services(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.Pod:
		realObj, err := cs.CoreClient.CoreV1().Pods(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.Secret:
		realObj, err := cs.CoreClient.CoreV1().Secrets(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ext_v1beta.Deployment:
		realObj, err := cs.CoreClient.ExtensionsV1beta1().Deployments(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_v1beta.Deployment:
		realObj, err := cs.CoreClient.AppsV1beta1().Deployments(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *apps_v1.Deployment:
		realObj, err := cs.CoreClient.AppsV1().Deployments(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_v2alpha1.CronJob:
		realObj, err := cs.CoreClient.BatchV2alpha1().CronJobs(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_ext_v1beta.DaemonSet:
		realObj, err := cs.CoreClient.ExtensionsV1beta1().DaemonSets(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.ConfigMap:
		realObj, err := cs.CoreClient.CoreV1().ConfigMaps(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.ServiceAccount:
		realObj, err := cs.CoreClient.CoreV1().ServiceAccounts(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko_ext_v1beta.Ingress:
		realObj, err := cs.CoreClient.ExtensionsV1beta1().Ingresses(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.PersistentVolumeClaim:
		realObj, err := cs.CoreClient.CoreV1().PersistentVolumeClaims(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *ko.PersistentVolume:
		realObj, err := cs.CoreClient.CoreV1().PersistentVolumes().Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *rbac_v1beta.ClusterRole:
		realObj, err := cs.CoreClient.RbacV1beta1().ClusterRoles().Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *rbac_v1beta.ClusterRoleBinding:
		realObj, err := cs.CoreClient.RbacV1beta1().ClusterRoleBindings().Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *rbac_v1.ClusterRole:
		realObj, err := cs.CoreClient.RbacV1().ClusterRoles().Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *rbac_v1.ClusterRoleBinding:
		realObj, err := cs.CoreClient.RbacV1().ClusterRoleBindings().Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *agg_v1betaObj.APIService:
		realObj, err := cs.APIRegClientV1beta1.APIServices().Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *rbac_v1beta.RoleBinding:
		realObj, err := cs.CoreClient.RbacV1beta1().RoleBindings(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	case *apps_v1.StatefulSet:
		realObj, err := cs.CoreClient.AppsV1().StatefulSets(metadata.Namespace).Get(metadata.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	default:
	}
	if sc, ok := obj.(*v1.StorageClass); ok {
		realObj, err := cs.CoreClient.StorageV1().StorageClasses().Get(sc.Name, meta_v1.GetOptions{ResourceVersion: "0"})
		if err != nil {
			return false, err
		}
		return realObj != nil, nil
	}
	return false, errors.New("Unsupported Kubernetes runtime.object kind: " + obj.GetObjectKind().GroupVersionKind().Kind)
}
