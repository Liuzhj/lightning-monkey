package strategies

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"sort"
	"strings"
	"sync"
	"time"
)

type ClusterStatementStrategy interface {
	CanDeployETCD() entities.ConditionCheckedResult
	CanDeployMasterComponents() entities.ConditionCheckedResult
	CanDeployMinion() entities.ConditionCheckedResult
	GetETCDNodeAddresses() []string
	GetMasterNodeAddresses() []string
	GetAgent(metadataId string) (*entities.Agent, error)
	GetAgents() []interface{}
	Load(cluster *entities.Cluster, agents []*entities.Agent)
	UpdateCache(agents []*entities.Agent)
}

type DefaultClusterStatementStrategy struct {
	lockObj                         *sync.RWMutex
	cluster                         *entities.Cluster
	agents                          map[string] /*Agent Role*/ map[string] /*Agent Status*/ []*entities.Agent
	agents_id_indexes               map[string] /*Metadata ID*/ *entities.Agent
	TotalETCDAgentCount             int
	TotalMasterAgentCount           int
	TotalMinionAgentCount           int
	TotalProvisionedETCDNodeCount   int
	TotalProvisionedMasterNodeCount int
}

func (ds *DefaultClusterStatementStrategy) Load(cluster *entities.Cluster, agents []*entities.Agent) {
	ds.cluster = cluster
	if ds.agents == nil {
		ds.agents = make(map[string]map[string][]*entities.Agent)
	}
	if ds.agents_id_indexes == nil {
		ds.agents_id_indexes = make(map[string]*entities.Agent)
	}
	if ds.lockObj == nil {
		ds.lockObj = &sync.RWMutex{}
	}
	if agents == nil || len(agents) == 0 {
		return
	}
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
func (ds *DefaultClusterStatementStrategy) GetETCDNodeAddresses() []string {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	agents := ds.agents[entities.AgentRole_ETCD]
	ips := []string{}
	if agents == nil {
		return ips
	}
	for _, as := range agents {
		if as == nil || len(as) == 0 {
			continue
		}
		for i := 0; i < len(as); i++ {
			ips = append(ips, as[i].LastReportIP)
		}
	}
	sort.Strings(ips)
	return ips
}

func (ds *DefaultClusterStatementStrategy) GetMasterNodeAddresses() []string {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	agents := ds.agents[entities.AgentRole_Master]
	ips := []string{}
	if agents == nil {
		return ips
	}
	for status, as := range agents {
		if as == nil || len(as) == 0 {
			continue
		}
		if strings.ToLower(status) != strings.ToLower(entities.AgentStatus_Running) {
			continue
		}
		for i := 0; i < len(as); i++ {
			ips = append(ips, as[i].LastReportIP)
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

func (ds *DefaultClusterStatementStrategy) GetAgents() []interface{} {
	ds.lockObj.RLock()
	defer ds.lockObj.RUnlock()
	arr := make([]interface{}, 0, len(ds.agents_id_indexes))
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
	ds.lockObj.Lock()
	defer ds.lockObj.Unlock()
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
	ds.lockObj.Lock()
	defer ds.lockObj.Unlock()
	ds.agents_id_indexes[agent.MetadataId] = agent
	var agentRoles []string
	if agent.HasMasterRole {
		agentRoles = append(agentRoles, entities.AgentRole_Master)
		if time.Since(agent.LastReportTime).Seconds() <= 30 && (agent.LastReportStatus == entities.AgentStatus_Running || agent.LastReportStatus == entities.AgentStatus_Provisioning) {
			ds.TotalMasterAgentCount++
			if agent.HasProvisionedMasterComponents {
				ds.TotalProvisionedMasterNodeCount++
			}
		}
	}
	if agent.HasETCDRole {
		agentRoles = append(agentRoles, entities.AgentRole_ETCD)
		if time.Since(agent.LastReportTime).Seconds() <= 30 && (agent.LastReportStatus == entities.AgentStatus_Running || agent.LastReportStatus == entities.AgentStatus_Provisioning) {
			ds.TotalETCDAgentCount++
			if agent.HasProvisionedETCD {
				ds.TotalProvisionedETCDNodeCount++
			}
		}
	}
	if agent.HasMinionRole {
		agentRoles = append(agentRoles, entities.AgentRole_Minion)
		if time.Since(agent.LastReportTime).Seconds() <= 30 && (agent.LastReportStatus == entities.AgentStatus_Running || agent.LastReportStatus == entities.AgentStatus_Provisioning) {
			ds.TotalMinionAgentCount++
		}
	}
	for j := 0; j < len(agentRoles); j++ {
		var isOK bool
		var as map[string][]*entities.Agent
		var m []*entities.Agent
		if as, isOK = ds.agents[strings.ToLower(agentRoles[j])]; !isOK {
			as = make(map[string][]*entities.Agent)
		}
		if m, isOK = as[strings.ToLower(agent.LastReportStatus)]; !isOK {
			m = []*entities.Agent{}
		}
		m = append(m, agent)
		as[strings.ToLower(agent.LastReportStatus)] = m
		ds.agents[strings.ToLower(agentRoles[j])] = as
	}
}
