package strategies

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"sort"
	"strings"
	"time"
)

type ClusterStatementStrategy interface {
	CanDeployETCD() entities.ConditionCheckedResult
	CanDeployMasterComponents() entities.ConditionCheckedResult
	CanDeployMinion() entities.ConditionCheckedResult
	GetETCDNodeAddresses() []string
}

type DefaultClusterStatementStrategy struct {
	cluster                         *entities.Cluster
	agents                          map[string] /*Agent Role*/ map[string] /*Agent Status*/ []*entities.Agent
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
	if agents == nil || len(agents) == 0 {
		return
	}
	for i := 0; i < len(agents); i++ {
		agent := agents[i]
		var agentRoles []string
		if agent.HasMasterRole {
			agentRoles = append(agentRoles, entities.AgentRole_Master)
			if time.Since(agent.LastReportTime).Seconds() <= 30 && agent.LastReportStatus == entities.AgentStatus_Running {
				ds.TotalMasterAgentCount++
				if agent.HasProvisionedMasterComponents {
					ds.TotalProvisionedMasterNodeCount++
				}
			}
		}
		if agent.HasETCDRole {
			agentRoles = append(agentRoles, entities.AgentRole_ETCD)
			if time.Since(agent.LastReportTime).Seconds() <= 30 && agent.LastReportStatus == entities.AgentStatus_Running {
				ds.TotalETCDAgentCount++
				if agent.HasProvisionedETCD {
					ds.TotalProvisionedETCDNodeCount++
				}
			}
		}
		if agent.HasMinionRole {
			agentRoles = append(agentRoles, entities.AgentRole_Minion)
			if time.Since(agent.LastReportTime).Seconds() <= 30 && agent.LastReportStatus == entities.AgentStatus_Running {
				ds.TotalMinionAgentCount++
			}
		}
		for j := 0; j < len(agentRoles); j++ {
			var isOK bool
			var as map[string][]*entities.Agent
			var m []*entities.Agent
			if as, isOK = ds.agents[strings.ToLower(agentRoles[j])]; !isOK {
				as = make(map[string][]*entities.Agent)
				ds.agents[strings.ToLower(agentRoles[j])] = as
			}
			if m, isOK = as[strings.ToLower(agent.LastReportStatus)]; !isOK {
				m = []*entities.Agent{}
				as[strings.ToLower(agent.LastReportStatus)] = m
			}
			m = append(m, agent)
		}
	}
}

func (ds *DefaultClusterStatementStrategy) CanDeployETCD() entities.ConditionCheckedResult {
	if ds.cluster.ExpectedETCDCount == 0 {
		return entities.ConditionInapplicable
	}
	if ds.TotalETCDAgentCount != ds.cluster.ExpectedETCDCount {
		return entities.ConditionNotConfirmed
	}
	return entities.ConditionConfirmed
}

func (ds *DefaultClusterStatementStrategy) CanDeployMasterComponents() entities.ConditionCheckedResult {
	//ensures that there has enough ETCD nodes has been started before running Kubernetes API server.
	if ds.TotalMasterAgentCount >= 1 && (ds.TotalProvisionedETCDNodeCount >= ds.cluster.ExpectedETCDCount) {
		return entities.ConditionConfirmed
	}
	return entities.ConditionNotConfirmed
}

func (ds *DefaultClusterStatementStrategy) CanDeployMinion() entities.ConditionCheckedResult {
	if ds.TotalProvisionedMasterNodeCount >= 1 {
		return entities.ConditionConfirmed
	}
	return entities.ConditionNotConfirmed
}

//TODO: may occur errors while building an ETCD cluster.
func (ds *DefaultClusterStatementStrategy) GetETCDNodeAddresses() []string {
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
