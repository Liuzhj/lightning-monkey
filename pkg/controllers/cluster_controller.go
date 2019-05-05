package controllers

import (
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/g0194776/lightningmonkey/pkg/strategies"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ClusterController struct {
	onceObj           *sync.Once
	cluster           *entities.Cluster
	lockObj           *sync.RWMutex
	masterClient      *k8s.Clientset
	networkController NetworkStackController
	stopChan          chan int
	storageDriver     storage.StorageDriver
	strategy          strategies.ClusterStatementStrategy
}

func (cc *ClusterController) ProvisionedComponent(agentId string, role string) error {
	return cc.strategy.ProvisionedComponent(agentId, role)
}

func (cc *ClusterController) Initialize(storageDriver storage.StorageDriver, strategy strategies.ClusterStatementStrategy) {
	if cc.lockObj == nil {
		cc.lockObj = &sync.RWMutex{}
	}
	if cc.stopChan == nil {
		cc.stopChan = make(chan int)
	}
	cc.storageDriver = storageDriver
	cc.strategy = strategy
}

func (cc *ClusterController) CanDeployETCD() entities.ConditionCheckedResult {
	return cc.strategy.CanDeployETCD()
}

func (cc *ClusterController) CanDeployMasterComponents() entities.ConditionCheckedResult {
	return cc.strategy.CanDeployMasterComponents()
}

func (cc *ClusterController) CanDeployMinion() entities.ConditionCheckedResult {
	return cc.strategy.CanDeployMinion()
}

func (cc *ClusterController) Dispose() {
	if cc.stopChan != nil {
		close(cc.stopChan)
	}
}

func (cc *ClusterController) GetAgentsAddress(role string, mustStatusFlag entities.AgentStatusFlag) []string {
	return cc.strategy.GetAgentsAddress(role, mustStatusFlag)
}

func (cc *ClusterController) GetAgent(metadataId string) (*entities.Agent, error) {
	return cc.strategy.GetAgent(metadataId)
}

func (cc *ClusterController) GetAgents() []*entities.Agent {
	return cc.strategy.GetAgents()
}

func (cc *ClusterController) Load(cluster *entities.Cluster, agents []*entities.Agent) {
	cc.cluster = cluster
	if cc.onceObj == nil {
		cc.onceObj = &sync.Once{}
	}
	cc.strategy.Load(cluster, agents)
	go cc.updateClusterStatusProc()
}

func (cc *ClusterController) UpdateCache(agents []*entities.Agent) {
	cc.strategy.UpdateCache(agents)
}

func (cc *ClusterController) updateClusterStatusProc() {
	cc.onceObj.Do(func() {
		var err error
		for {
			logrus.Debugf("Cluster %s status check loop...", cc.cluster.Id.Hex())
			select {
			case _, isOpen := <-cc.stopChan:
				if !isOpen {
					return
				}
			default:
				//4 steps cluster status flow:
				//new -> provisioning -> ready -> available
				masterCount := len(cc.strategy.GetAgentsAddress(entities.AgentRole_Master, entities.AgentStatusFlag_Provisioned))
				minionCount := len(cc.strategy.GetAgentsAddress(entities.AgentRole_Minion, entities.AgentStatusFlag_Provisioned))
				logrus.Debugf("Cluster %s, Master=%d, Minion=%d", cc.cluster.Id.Hex(), masterCount, minionCount)
				if masterCount <= 0 {
					if cc.cluster.Status != entities.ClusterNew {
						cc.cluster.Status = entities.ClusterUncontrollable
					}
				} else if (masterCount > 0 && minionCount <= 0) || (!cc.cluster.HasProvisionedNetworkStack) {
					cc.cluster.Status = entities.ClusterProvisioning
				} else if masterCount > 0 && minionCount > 0 && cc.cluster.HasProvisionedNetworkStack {
					cc.cluster.Status = entities.ClusterReady
				}
				//provisioning Kubernetes network stack.
				if cc.cluster.Status == entities.ClusterProvisioning && !cc.cluster.HasProvisionedNetworkStack {
					err = cc.deployNetworkStack()
					if err != nil {
						logrus.Errorf("Failed to deploy Kubernetes network stack for cluster: %s, reason: %s", cc.cluster.Id.Hex(), err.Error())
					} else {
						cc.cluster.HasProvisionedNetworkStack = true
						cc.cluster.NetworkStackProvisionTime = time.Now()
					}
				}
				//save cluster newest information to remote database.
				err = cc.storageDriver.UpdateCluster(cc.cluster)
				if err != nil {
					logrus.Error(err)
				}
			}
			time.Sleep(time.Second * 5)
		}
	})
}

func (cc *ClusterController) deployNetworkStack() error {
	err := cc.ensureMasterConnection()
	if err != nil {
		return fmt.Errorf("Failed to ensure an available Kubernetes client instance, error: %s", err.Error())
	}
	if cc.networkController == nil {
		cc.networkController, err = CreateNetworkStackController(cc.masterClient, cc.cluster)
		if err != nil {
			return err
		}
	}
	return cc.networkController.Install()
}

func (cc *ClusterController) ensureMasterConnection() error {
	if cc.masterClient != nil {
		return nil
	}
	certContent, err := cc.getMasterCerficiate()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/tmp/kubernetes-certs/%s/admin.conf", uuid.NewV4().String())
	err = os.MkdirAll(filepath.Dir(path), 0644)
	if err != nil {
		return fmt.Errorf("Failed to create temporary path for writing Kubernetes certificate: %s, error: %s", filepath.Dir(path), err.Error())
	}
	defer func() {
		//clean resource.
		_ = os.RemoveAll(filepath.Dir(path))
	}()
	err = ioutil.WriteFile(path, []byte(certContent), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write Kubernetes certificate content, error: %s", err.Error())
	}
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return fmt.Errorf("Failed to initialize Kubernetes certificate content client for cluster: %s, error: %s", cc.cluster.Id.Hex(), err.Error())
	}
	cc.masterClient, err = k8s.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to initialize Kubernetes client for cluster: %s, error: %s", cc.cluster.Id.Hex(), err.Error())
	}
	return nil
}

func (cc *ClusterController) getMasterCerficiate() (string, error) {
	addresses := cc.GetAgentsAddress(entities.AgentRole_Master, entities.AgentStatusFlag_Provisioned)
	if addresses == nil || len(addresses) == 0 {
		return "", errors.New("No any available master nodes.")
	}
	certName := fmt.Sprintf("%s/admin.conf", addresses[0])
	cert, err := cc.storageDriver.GetCertificatesByClusterIdAndName(cc.cluster.Id.Hex(), certName)
	if err != nil {
		return "", fmt.Errorf("Failed to get certificiate content with given name: %s, error: %s", certName, err.Error())
	}
	if cert == nil || cert.Content == "" {
		return "", fmt.Errorf("Certificate name: \"%s\" not found.", certName)
	}
	return cert.Content, nil
}
