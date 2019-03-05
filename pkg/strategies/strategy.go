package strategies

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"strings"
)

type ClusterStatementStrategy interface {
	CanDeployETCD() bool
	CanDeployMasterComponents() bool
	CanDeployMinion() bool
}

type DefaultClusterStatementStrategy struct {
	cluster *entities.Cluster
	agents  map[string] /*Agent Role*/ map[string] /*Agent Status*/ []*entities.Agent
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
		for j := 0; j < len(agent.Roles); j++ {
			var isOK bool
			var as map[string][]*entities.Agent
			var m []*entities.Agent
			if as, isOK = ds.agents[strings.ToLower(agent.Roles[j])]; !isOK {
				as = make(map[string][]*entities.Agent)
				ds.agents[strings.ToLower(agent.Roles[j])] = as
			}
			if m, isOK = as[strings.ToLower(agent.LastReportStatus)]; !isOK {
				m = []*entities.Agent{}
				as[strings.ToLower(agent.LastReportStatus)] = m
			}
			m = append(m, agent)
		}
	}
}

func (ds *DefaultClusterStatementStrategy) CanDeployMasterComponents() bool {
	return false
}

func (ds *DefaultClusterStatementStrategy) CanDeployETCD() bool {
	return false
}

func (ds *DefaultClusterStatementStrategy) CanDeployMinion() bool {
	return false
}
