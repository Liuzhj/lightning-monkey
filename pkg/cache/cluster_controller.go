package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/controllers"
	"github.com/g0194776/lightningmonkey/pkg/controllers/dns"
	"github.com/g0194776/lightningmonkey/pkg/controllers/network"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/g0194776/lightningmonkey/pkg/monitors"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	agg_v1beta "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1beta1"
)

//go:generate mockgen -package=mock_lm -destination=../../mocks/mock_cluster_controller.go -source=cluster_controller.go ClusterController
type ClusterController interface {
	Dispose() //clean all of in use resource including backend watching jobs.
	GetAgentFromETCD(agentId string) (*entities.LightningMonkeyAgent, error)
	GetSynchronizedRevision() int64
	GetStatus() string
	GetClusterId() string
	GetCertificates() entities.LightningMonkeyCertificateCollection
	GetNextJob(agent entities.LightningMonkeyAgent, updateAgentDeploymentPhase func(int)) (entities.AgentJob, error)
	GetTotalCountByRole(role string) int
	GetTotalProvisionedCountByRole(role string) int
	GetSettings() entities.LightningMonkeyClusterSettings
	GetCachedAgent(agentId string) (*entities.LightningMonkeyAgent, error)
	Initialize(sd storage.LightningMonkeyStorageDriver)
	SetSynchronizedRevision(id int64)
	SetCancellationFunc(f func()) //used for disposing in use resource.
	Lock()
	UnLock()
	UpdateClusterSettings(settings entities.LightningMonkeyClusterSettings) ClusterController
	OnAgentChanged(agent entities.LightningMonkeyAgent, isDeleted bool) error
	OnCertificateChanged(name string, cert string, isDeleted bool) error
	InitializeKubernetesClient() error
	InitializeNetworkController() error
	InitializeDNSController() error
	InitializeExtensionDeploymentController() error
	GetNetworkController() controllers.DeploymentController
	GetDNSController() controllers.DeploymentController
	GetExtensionDeploymentController() controllers.DeploymentController
	GetWachPoints() []entities.WatchPoint
	GetRandomAdminConfFromMasterAgents() (string, error)
	GetNodesInformation() ([]entities.KubernetesNodeInfo, error)
	EnableMonitors()
	GetAgentList(onlineOnly bool) ([]entities.LightningMonkeyAgentBriefInformation, error)
}

type ClusterControllerImple struct {
	cs                   *k8s.KubernetesClientSet
	k8sClientIP          string
	cache                *AgentCache
	certs                map[string]string
	cancellationFunc     func()
	monitors             []monitors.KubernetesResourceMonitor
	jobScheduler         ClusterJobScheduler
	isDisposed           uint32
	lockObj              *sync.Mutex
	monitorLockObj       *sync.Mutex
	nsc                  controllers.DeploymentController
	ddc                  controllers.DeploymentController
	edc                  controllers.DeploymentController
	settings             entities.LightningMonkeyClusterSettings
	synchronizedRevision int64
	sd                   storage.LightningMonkeyStorageDriver
}

func (cc *ClusterControllerImple) GetSettings() entities.LightningMonkeyClusterSettings {
	return cc.settings
}

//used for debugging internal state.
func (cc *ClusterControllerImple) GetTotalCountByRole(role string) int {
	return cc.cache.GetTotalCountByRole(role)
}

//used for debugging internal state.
func (cc *ClusterControllerImple) GetTotalProvisionedCountByRole(role string) int {
	return cc.cache.GetTotalProvisionedCountByRole(role)
}

func (cc *ClusterControllerImple) Initialize(sd storage.LightningMonkeyStorageDriver) {
	if cc.lockObj == nil {
		cc.lockObj = &sync.Mutex{}
	}
	if cc.monitorLockObj == nil {
		cc.monitorLockObj = &sync.Mutex{}
	}
	cc.sd = sd
	cc.certs = make(map[string]string)
	cc.cache = &AgentCache{}
	cc.cache.Initialize()
	cc.jobScheduler = &ClusterJobSchedulerImple{}
	cc.jobScheduler.InitializeStrategies()
	err := cc.fullSync(cc.sd)
	if err != nil {
		logrus.Fatalf("Failed to full-sync cluster %s data, error: %s", cc.settings.Id, err.Error())
		os.Exit(1)
	}
}

func (cc *ClusterControllerImple) fullSync(sd storage.LightningMonkeyStorageDriver) error {
	rsp, err := sd.Get(context.Background(), fmt.Sprintf("/lightning-monkey/clusters/%s/", cc.GetClusterId()), clientv3.WithPrefix())
	if err != nil {
		return err
	}
	if rsp.Count == 0 {
		return nil
	}
	var subKeys []string
	for i := 0; i < len(rsp.Kvs); i++ {
		subKeys = strings.FieldsFunc(string(rsp.Kvs[i].Key), func(c rune) bool {
			return c == '/'
		})
		if certKey, isChanged := isCertificatesChanged(subKeys); isChanged {
			cert := string(rsp.Kvs[i].Value)
			if cert == "" {
				logrus.Errorf("Illegal certificate content being received from remote ETCD event, key: %s", string(rsp.Kvs[i].Key))
				continue
			}
			logrus.Debugf("Collecting generated certificate for cluster: %s, cert-name: %s", cc.GetClusterId(), certKey)
			cc.Lock()
			err = cc.OnCertificateChanged(certKey, cert, false)
			cc.UnLock()
			if err != nil {
				logrus.Errorf("Failed to update hot cache with certificate changes, cluster: %s, key: %s error: %s", cc.GetClusterId(), string(rsp.Kvs[i].Key), err.Error())
				continue
			}
		}
	}
	return nil
}

func (cc *ClusterControllerImple) Dispose() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("Occurred an unhandled exception during disposing cluster controller, cluster-id: %s, error: %v", cc.settings.Id, err)
		}
	}()
	atomic.StoreUint32(&cc.isDisposed, 1)
	if cc.cancellationFunc != nil {
		cc.cancellationFunc()
	}
	cc.cache = nil
}

func (cc *ClusterControllerImple) GetSynchronizedRevision() int64 {
	return cc.synchronizedRevision
}

func (cc *ClusterControllerImple) GetStatus() string {
	return entities.ClusterReady
}

func (cc *ClusterControllerImple) GetClusterId() string {
	return cc.settings.Id
}

func (cc *ClusterControllerImple) GetCertificates() entities.LightningMonkeyCertificateCollection {
	collection := make([]*entities.CertificateKeyPair, len(cc.certs))
	i := 0
	for k, v := range cc.certs {
		collection[i] = &entities.CertificateKeyPair{Name: strings.Replace(k, "_", "/", -1), Value: v}
		i++
	}
	return collection
}

func (cc *ClusterControllerImple) GetNextJob(agent entities.LightningMonkeyAgent, updateAgentDeploymentPhase func(int)) (entities.AgentJob, error) {
	return cc.jobScheduler.GetNextJob(cc, agent, cc.cache, updateAgentDeploymentPhase)
}

func (cc *ClusterControllerImple) SetSynchronizedRevision(id int64) {
	cc.synchronizedRevision = id
}

func (cc *ClusterControllerImple) SetCancellationFunc(f func()) {
	cc.cancellationFunc = f
}

func (cc *ClusterControllerImple) Lock() {
	cc.lockObj.Lock()
}

func (cc *ClusterControllerImple) UnLock() {
	cc.lockObj.Unlock()
}

func (cc *ClusterControllerImple) OnAgentChanged(agent entities.LightningMonkeyAgent, isDeleted bool) error {
	if atomic.LoadUint32(&cc.isDisposed) == 1 {
		return fmt.Errorf("Cannot update cache to a disposed cluster controller, cluster-id: %s", cc.settings.Id)
	}
	if isDeleted || agent.State == nil {
		cc.cache.Offline(agent)
	} else {
		cc.cache.Online(agent)
	}
	return nil
}

func (cc *ClusterControllerImple) OnCertificateChanged(name string, cert string, isDeleted bool) error {
	logrus.Debugf("Certificate %s Changed: is-deleted: %t", name, isDeleted)
	logrus.Debugf("Certificate %s value: %s", name, cert)
	if atomic.LoadUint32(&cc.isDisposed) == 1 {
		return fmt.Errorf("Cannot update cache to a disposed cluster controller, cluster-id: %s", cc.settings.Id)
	}
	if isDeleted {
		delete(cc.certs, name)
		return nil
	}
	cc.certs[name] = cert
	return nil
}

func (cc *ClusterControllerImple) UpdateClusterSettings(settings entities.LightningMonkeyClusterSettings) ClusterController {
	cc.settings = settings
	return cc
}

func (cc *ClusterControllerImple) GetCachedAgent(agentId string) (*entities.LightningMonkeyAgent, error) {
	if atomic.LoadUint32(&cc.isDisposed) == 1 {
		return nil, fmt.Errorf("Cannot update cache to a disposed cluster controller, cluster-id: %s", cc.settings.Id)
	}
	cc.cache.Lock()
	defer cc.cache.Unlock()
	var isOK bool
	var agent *entities.LightningMonkeyAgent
	if agent, isOK = cc.cache.etcd[agentId]; isOK {
		return agent, nil
	}
	if agent, isOK = cc.cache.k8sMaster[agentId]; isOK {
		return agent, nil
	}
	if agent, isOK = cc.cache.k8sMinion[agentId]; isOK {
		return agent, nil
	}
	if agent, isOK = cc.cache.ha[agentId]; isOK {
		return agent, nil
	}
	if agent, isOK = cc.cache.pool[agentId]; isOK {
		return agent, nil
	}
	return nil, nil
}

func (cc *ClusterControllerImple) InitializeKubernetesClient() error {
	if cc.cs == nil {
		cc.cs = &k8s.KubernetesClientSet{}
	}
	if cc.cs.CoreClient != nil {
		return nil
	}
	agent := cc.cache.GetFirstProvisionedKubernetesMasterAgent()
	if agent == nil {
		return fmt.Errorf("CANNOT retrieve any agent which provisioned Kubernetes master on cluster: %s", cc.GetClusterId())
	}
	if agent.AdminCertificate == "" {
		return fmt.Errorf("Illegal administrative certificate on agent(%s), It's empty at all. Please consider report a BUG to the community.", agent.Id)
	}
	//initialize Kubernetes client by pre-generated administrative certificate.
	adminCertPath := "/etc/kubernetes/admin"
	_ = os.MkdirAll(adminCertPath, 0644)
	filePath := filepath.Join(adminCertPath, fmt.Sprintf("%s.yml", cc.GetClusterId()))
	//clean old file.
	_ = os.RemoveAll(filePath)
	err := ioutil.WriteFile(filePath, []byte(agent.AdminCertificate), 0644) //rw-r-r
	if err != nil {
		return fmt.Errorf("Failed to write Kubernetes master client configuration to local disk, error: %s", err.Error())
	}
	config, err := clientcmd.BuildConfigFromFlags("", filePath)
	if err != nil {
		return fmt.Errorf("Failed to build Kubernetes master client configuration, error: %s", err.Error())
	}
	//reset rate limiter for larger cluster size.
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(25, 50)
	//create core client.
	cc.cs.CoreClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to initialize core Kubernetes client, error: %s", err.Error())
	}
	//create particular client for API Service.
	cc.cs.APIRegClientV1beta1, err = agg_v1beta.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to initialize particular Kubernetes client for API reg, error: %s", err.Error())
	}
	cc.k8sClientIP = agent.State.LastReportIP
	return nil
}

func (cc *ClusterControllerImple) InitializeNetworkController() error {
	if cc.nsc != nil {
		return nil
	}
	var err error
	cc.nsc, err = network.CreateNetworkStackController(cc.cs, cc.k8sClientIP, cc.GetSettings())
	if err != nil {
		return fmt.Errorf("Failed to initialize Kubernetes network stack controller on cluster: %s, error: %s", cc.GetClusterId(), err.Error())
	}
	return nil
}

func (cc *ClusterControllerImple) GetNetworkController() controllers.DeploymentController {
	return cc.nsc
}

func (cc *ClusterControllerImple) InitializeDNSController() error {
	if cc.ddc != nil {
		return nil
	}
	var err error
	cc.ddc, err = dns.CreateDNSDeploymentController(cc.cs, cc.k8sClientIP, cc.GetSettings())
	if err != nil {
		return fmt.Errorf("Failed to initialize Kubernetes DNS deployment controller on cluster: %s, error: %s", cc.GetClusterId(), err.Error())
	}
	return nil
}

func (cc *ClusterControllerImple) GetDNSController() controllers.DeploymentController {
	return cc.ddc
}

func (cc *ClusterControllerImple) GetWachPoints() []entities.WatchPoint {
	if cc.monitors == nil || len(cc.monitors) == 0 {
		return nil
	}
	wps := []entities.WatchPoint{}
	for i := 0; i < len(cc.monitors); i++ {
		wpl := cc.monitors[i].GetWatchPoints()
		if wpl == nil || len(wpl) == 0 {
			continue
		}
		wps = append(wps, wpl...)
	}
	return wps
}

func (cc *ClusterControllerImple) EnableMonitors() {
	cc.monitorLockObj.Lock()
	defer cc.monitorLockObj.Unlock()
	if cc.monitors != nil && len(cc.monitors) > 0 {
		return
	}
	logrus.Debugf("Enabling monitors to cluster: %s...", cc.GetClusterId())
	cc.monitors = []monitors.KubernetesResourceMonitor{}
	//System Component.
	sysMonitor := monitors.NewMonitor("sys", cc.cs, cc.GetClusterId())
	err := sysMonitor.Start()
	if err != nil {
		logrus.Errorf("Failed to start Kubernetes system component monitor, error: %s", err.Error())
		return
	}
	cc.monitors = append(cc.monitors, sysMonitor)
	//Kubernetes Deployment.
	deployMonitor := monitors.NewMonitor("deployment", cc.cs, cc.GetClusterId())
	err = deployMonitor.Start()
	if err != nil {
		logrus.Errorf("Failed to start Kubernetes deployment monitor, error: %s", err.Error())
		return
	}
	cc.monitors = append(cc.monitors, deployMonitor)
	//Kubernetes Daemonset.
	dsMonitor := monitors.NewMonitor("daemonset", cc.cs, cc.GetClusterId())
	err = dsMonitor.Start()
	if err != nil {
		logrus.Errorf("Failed to start Kubernetes daemonset monitor, error: %s", err.Error())
		return
	}
	cc.monitors = append(cc.monitors, dsMonitor)
}

func (cc *ClusterControllerImple) GetExtensionDeploymentController() controllers.DeploymentController {
	return cc.edc
}

func (cc *ClusterControllerImple) InitializeExtensionDeploymentController() error {
	if cc.edc != nil {
		return nil
	}
	var err error
	cc.edc, err = controllers.CreateExtensionDeploymentController(cc.cs, cc.k8sClientIP, cc.GetSettings())
	if err != nil {
		return fmt.Errorf("Failed to initialize extension deployment controller on cluster: %s, error: %s", cc.GetClusterId(), err.Error())
	}
	return nil
}

func (cc *ClusterControllerImple) GetRandomAdminConfFromMasterAgents() (string, error) {
	return cc.cache.GetAdminConfFromMasterAgents(), nil
}

func (cc *ClusterControllerImple) GetNodesInformation() ([]entities.KubernetesNodeInfo, error) {
	if cc.cs == nil || cc.cs.CoreClient == nil {
		return nil, fmt.Errorf("Cluster %s internal Kubernetes client has not be initialized yet!", cc.GetSettings().Id)
	}
	ns, err := cc.cs.CoreClient.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to list cluster(%s)'s node, error: %s", cc.GetSettings().Id, err.Error())
	}
	if ns == nil || ns.Items == nil || len(ns.Items) == 0 {
		return []entities.KubernetesNodeInfo{}, nil
	}
	result := []entities.KubernetesNodeInfo{}
	for i := 0; i < len(ns.Items); i++ {
		if ns.Items[i].Status.Addresses == nil || len(ns.Items[i].Status.Addresses) == 0 {
			continue
		}
		var nodeIP string
		for _, v := range ns.Items[i].Status.Addresses {
			if v.Type == v1.NodeInternalIP {
				nodeIP = v.Address
				break
			}
		}
		if nodeIP == "" {
			continue
		}
		result = append(result, entities.KubernetesNodeInfo{
			NodeIP:  nodeIP,
			PodCIDR: ns.Items[i].Spec.PodCIDR,
		})
	}
	return result, nil
}

func (cc *ClusterControllerImple) GetAgentList(onlineOnly bool) ([]entities.LightningMonkeyAgentBriefInformation, error) {
	if onlineOnly {
		//TODO(g0194776): to support it by using in-process cache.
		return nil, fmt.Errorf("Not supported listing only online agents from cluster: %s", cc.GetSettings().Id)
	}
	//fully list all keys what under this cluster.
	rsp, err := cc.sd.Get(context.Background(), fmt.Sprintf("/lightning-monkey/clusters/%s/", cc.GetClusterId()), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if rsp.Count == 0 {
		return nil, nil
	}
	var subKeys []string
	agents := []entities.LightningMonkeyAgentBriefInformation{}
	for i := 0; i < len(rsp.Kvs); i++ {
		subKeys = strings.FieldsFunc(string(rsp.Kvs[i].Key), func(c rune) bool {
			return c == '/'
		})
		//detect agents changes.
		if agentId, changed := isAgentSettingsChanged(subKeys); changed {
			agent, err := cc.GetAgentFromETCD(agentId)
			if err != nil {
				logrus.Errorf("Failed to retrieve newest version of Lightning Monkey's Agent data from remote ETCD, error: %s", err.Error())
				continue
			}
			if agent == nil {
				logrus.Errorf("Failed to retrieve newest version of Lightning Monkey's Agent data from remote ETCD, error: agent %s not found in the cluster %s", agentId, cc.GetClusterId())
				continue
			}
			a := entities.LightningMonkeyAgentBriefInformation{
				Id:              agent.Id,
				HasETCDRole:     agent.HasETCDRole,
				HasMasterRole:   agent.HasMasterRole,
				HasMinionRole:   agent.HasMinionRole,
				HasHARole:       agent.HasHARole,
				Hostname:        agent.Hostname,
				HostInformation: agent.HostInformation,
				DeploymentPhase: agent.DeploymentPhase,
			}
			if agent.State != nil {
				as := *agent.State
				a.State = &as
			}
			agents = append(agents, a)
		}
	}
	return agents, nil
}

func (cc *ClusterControllerImple) GetAgentFromETCD(agentId string) (*entities.LightningMonkeyAgent, error) {
	if agentId == "" {
		return nil, nil
	}
	agent := entities.LightningMonkeyAgent{}
	settingsPath := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/settings", cc.GetClusterId(), agentId)
	ctx, cancel := context.WithTimeout(context.Background(), cc.sd.GetRequestTimeoutDuration())
	defer cancel()
	rsp, err := cc.sd.Get(ctx, settingsPath)
	if err != nil {
		return nil, err
	}
	if rsp.Count == 0 {
		return nil, nil
	}
	err = json.Unmarshal(rsp.Kvs[0].Value, &agent)
	if err != nil {
		return nil, err
	}
	statePath := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/state", cc.GetClusterId(), agentId)
	ctx2, cancel2 := context.WithTimeout(context.Background(), cc.sd.GetRequestTimeoutDuration())
	defer cancel2()
	rsp, err = cc.sd.Get(ctx2, statePath)
	if err != nil {
		return nil, err
	}
	if rsp.Count == 0 {
		//ignored missed state object, it's lease-guaranteed.
		return &agent, nil
	}
	state := entities.AgentState{}
	err = json.Unmarshal(rsp.Kvs[0].Value, &state)
	if err != nil {
		return nil, err
	}
	agent.State = &state
	return &agent, nil
}
