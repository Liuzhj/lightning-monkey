package controllers

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/g0194776/lightningmonkey/pkg/strategies"
	"github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"
	"sync"
	"time"
)

type ClusterController struct {
	onceObj       *sync.Once
	cluster       *entities.Cluster
	lockObj       *sync.RWMutex
	masterClient  *k8s.Clientset
	stopChan      chan int
	storageDriver storage.StorageDriver
	strategy      strategies.ClusterStatementStrategy
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

func (cc *ClusterController) GetAgentsAddress(role string, mustRunningStatus bool) []string {
	return cc.strategy.GetAgentsAddress(role, mustRunningStatus)
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

func (cc *ClusterController) ensureMasterConnection() error {
	return nil
}

func (cc *ClusterController) updateClusterStatusProc() {
	cc.onceObj.Do(func() {
		var err error
		for {
			select {
			case _, isOpen := <-cc.stopChan:
				if !isOpen {
					return
				}
			default:
				//4 steps cluster status flow:
				//new -> provisioning -> ready -> available
				masterCount := len(cc.strategy.GetAgentsAddress(entities.AgentRole_Master, true))
				minionCount := len(cc.strategy.GetAgentsAddress(entities.AgentRole_Minion, true))
				if masterCount <= 0 {
					cc.cluster.Status = entities.ClusterNew
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
			time.Sleep(time.Second * 3)
		}
	})
}

func (cc *ClusterController) deployNetworkStack() error {
	err := cc.ensureMasterConnection()
	if err != nil {
		return fmt.Errorf("Failed to ensure an available Kubernetes client instance, error: %s", err.Error())
	}
	return nil
}
