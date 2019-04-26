package strategies

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"sort"
	"sync"
)

type agentExp func(a *entities.Agent) bool
type ClusterStatementStrategy interface {
	CanDeployETCD() entities.ConditionCheckedResult
	CanDeployMasterComponents() entities.ConditionCheckedResult
	CanDeployMinion() entities.ConditionCheckedResult
	GetAgentsAddress(role string, mustStatusFlag entities.AgentStatusFlag) []string
	GetAgent(metadataId string) (*entities.Agent, error)
	GetAgents() []*entities.Agent
	Load(cluster *entities.Cluster, agents []*entities.Agent)
	UpdateCache(agents []*entities.Agent)
}

type DefaultClusterStatementStrategy struct {
	lockObj *sync.RWMutex
	cluster *entities.Cluster
	//agents                          map[string] /*Agent Role*/ map[string] /*Agent Status*/ []*entities.Agent
	agents_id_indexes               map[string] /*Metadata ID*/ *entities.Agent
	TotalETCDAgentCount             int
	TotalMasterAgentCount           int
	TotalMinionAgentCount           int
	TotalProvisionedETCDNodeCount   int
	TotalProvisionedMasterNodeCount int
}

func (ds *DefaultClusterStatementStrategy) Load(cluster *entities.Cluster, agents []*entities.Agent) {
	ds.cluster = cluster
	if ds.agents_id_indexes == nil {
		ds.agents_id_indexes = make(map[string]*entities.Agent)
	}
	if ds.lockObj == nil {
		ds.lockObj = &sync.RWMutex{}
	}
	if agents == nil || len(agents) == 0 {
		return
	}
	ds.lockObj.Lock()
	defer ds.lockObj.Unlock()
	//build index by role & metadata ID.
	for i := 0; i < len(agents); i++ {
		ds.addAgentToCache(agents[i])
	}
}

func (ds *DefaultClusterStatementStrategy) CanDeployETCD() entities.ConditionCheckedResult {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	if ds.cluster.ExpectedETCDCount == 0 {
		return entities.ConditionInapplicable
	}
	if ds.TotalETCDAgentCount != ds.cluster.ExpectedETCDCount {
		return entities.ConditionNotConfirmed
	}
	return entities.ConditionConfirmed
}

func (ds *DefaultClusterStatementStrategy) CanDeployMasterComponents() entities.ConditionCheckedResult {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	//ensures that there has enough ETCD nodes has been started before running Kubernetes API server.
	if ds.TotalMasterAgentCount >= 1 && (ds.TotalProvisionedETCDNodeCount >= ds.cluster.ExpectedETCDCount) {
		return entities.ConditionConfirmed
	}
	return entities.ConditionNotConfirmed
}

func (ds *DefaultClusterStatementStrategy) CanDeployMinion() entities.ConditionCheckedResult {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	if ds.TotalProvisionedMasterNodeCount >= 1 {
		return entities.ConditionConfirmed
	}
	return entities.ConditionNotConfirmed
}

//TODO: may occur errors while building an ETCD cluster.
func (ds *DefaultClusterStatementStrategy) GetAgentsAddress(role string, mustStatusFlag entities.AgentStatusFlag) []string {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	var expFunc agentExp
	switch role {
	case entities.AgentRole_ETCD:
		expFunc = func(a *entities.Agent) bool { return a.HasETCDRole }
	case entities.AgentRole_Master:
		expFunc = func(a *entities.Agent) bool { return a.HasMasterRole }
	case entities.AgentRole_Minion:
		expFunc = func(a *entities.Agent) bool { return a.HasMinionRole }
	default:
		//fast fail when occurs internal serious BUG.
		logrus.Fatalf("Illegal type of role name: %s", role)
		return nil
	}
	ips := []string{}
	for _, as := range ds.agents_id_indexes {
		if expFunc(as) {
			if mustStatusFlag == entities.AgentStatusFlag_Running /*running*/ && !as.IsRunning() {
				//unhealthy or report timed out.
				continue
			}
			if mustStatusFlag == entities.AgentStatusFlag_Provisioned /*provisioned*/ && !as.IsProvisioned() {
				//unhealthy or report timed out.
				continue
			}
			ips = append(ips, as.LastReportIP)
		}
	}
	sort.Strings(ips)
	return ips
}

func (ds *DefaultClusterStatementStrategy) GetAgent(metadataId string) (*entities.Agent, error) {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	agent := ds.agents_id_indexes[metadataId]
	if agent == nil {
		return nil, fmt.Errorf("Agent does not exist, Metadata-ID: %s", metadataId)
	}
	return agent, nil
}

func (ds *DefaultClusterStatementStrategy) GetAgents() []*entities.Agent {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	if len(ds.agents_id_indexes) == 0 {
		return nil
	}
	arr := make([]*entities.Agent, len(ds.agents_id_indexes))
	i := 0
	for _, agent := range ds.agents_id_indexes {
		arr[i] = agent
		i++
	}
	return arr
}

func (ds *DefaultClusterStatementStrategy) UpdateCache(agents []*entities.Agent) {
	ds.lockObj.Lock()
	defer ds.lockObj.Unlock()
	for i := 0; i < len(agents); i++ {
		a := agents[i]
		if orgAgent, existed := ds.agents_id_indexes[a.MetadataId]; existed {
			ds.updateAgentInternalStatus(orgAgent, a)
		} else {
			ds.addAgentToCache(a)
		}
	}
}

func (ds *DefaultClusterStatementStrategy) updateAgentInternalStatus(oldAgent, newAgent *entities.Agent) {
	if newAgent.HasProvisionedETCD && !oldAgent.HasProvisionedETCD {
		oldAgent.HasProvisionedETCD = true
		oldAgent.ETCDProvisionTime = newAgent.ETCDProvisionTime
	}
	if newAgent.HasProvisionedMasterComponents && !oldAgent.HasProvisionedMasterComponents {
		oldAgent.HasProvisionedMasterComponents = true
		oldAgent.MasterComponentsProvisionTime = newAgent.MasterComponentsProvisionTime
	}
	if newAgent.HasProvisionedMinion && !oldAgent.HasProvisionedMinion {
		oldAgent.HasProvisionedMinion = true
		oldAgent.MinionProvisionTime = newAgent.MinionProvisionTime
	}
}

func (ds *DefaultClusterStatementStrategy) addAgentToCache(agent *entities.Agent) {
	ds.agents_id_indexes[agent.MetadataId] = agent
	if !agent.IsRunning() {
		return
	}
	if agent.HasMasterRole {
		ds.TotalMasterAgentCount++
		if agent.HasProvisionedMasterComponents {
			ds.TotalProvisionedMasterNodeCount++
		}
	}
	if agent.HasETCDRole {
		ds.TotalETCDAgentCount++
		if agent.HasProvisionedETCD {
			ds.TotalProvisionedETCDNodeCount++
		}
	}
	if agent.HasMinionRole {
		ds.TotalMinionAgentCount++
	}
}
