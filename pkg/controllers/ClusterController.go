package controllers

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/g0194776/lightningmonkey/pkg/strategies"
	k8s "k8s.io/client-go/kubernetes"
	"sync"
)

type ClusterController struct {
	cluster       *entities.Cluster
	lockObj       *sync.RWMutex
	masterClient  *k8s.Clientset
	storageDriver storage.StorageDriver
	strategy      strategies.ClusterStatementStrategy
}

func (cc *ClusterController) Initialize(storageDriver storage.StorageDriver, strategy strategies.ClusterStatementStrategy) {
	if cc.lockObj == nil {
		cc.lockObj = &sync.RWMutex{}
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

func (cc *ClusterController) GetETCDNodeAddresses() []string {
	return cc.strategy.GetETCDNodeAddresses()
}

func (cc *ClusterController) GetMasterNodeAddresses() []string {
	return cc.strategy.GetMasterNodeAddresses()
}

func (cc *ClusterController) GetAgent(metadataId string) (*entities.Agent, error) {
	return cc.strategy.GetAgent(metadataId)
}

func (cc *ClusterController) GetAgents() []interface{} {
	return cc.strategy.GetAgents()
}

func (cc *ClusterController) Load(cluster *entities.Cluster, agents []*entities.Agent) {
	cc.cluster = cluster
	cc.strategy.Load(cluster, agents)
}

func (cc *ClusterController) UpdateCache(agents []*entities.Agent) {
	cc.strategy.UpdateCache(agents)
}

func (cc *ClusterController) ensureMasterConnection() error {

	return nil
}
