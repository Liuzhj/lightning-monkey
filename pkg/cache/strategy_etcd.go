package cache

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"strings"
)

type ClusterETCDJobStrategy struct {
}

func (js *ClusterETCDJobStrategy) GetStrategyName() string {
	return entities.AgentJob_Deploy_ETCD
}

func (js *ClusterETCDJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	clusterSettings := cc.GetSettings()
	//skipped, when satisfy expected count of ETCD
	if cache.GetTotalProvisionedCountByRole(entities.AgentRole_ETCD) >= clusterSettings.ExpectedETCDCount {
		return entities.ConditionInapplicable, "", nil, nil
	}
	if agent.HasETCDRole {
		if !agent.State.HasProvisionedETCD {
			//ensures that has enough nodes count of ETCD can continuously perform subsequent deployment task.
			if cache.GetTotalCountByRole(entities.AgentRole_ETCD) < clusterSettings.ExpectedETCDCount {
				return entities.ConditionNotConfirmed, "Waiting, Not equals required minimum count of ETCD nodes.", nil, nil
			}
			args := strings.Join(cache.GetAgentsAddress(entities.AgentRole_ETCD, entities.AgentStatusFlag_Whatever), ",")
			logrus.Debugf("Start dispatching ETCD deployment task to agent: %s, args: %s", agent.Id, args)
			return entities.ConditionConfirmed, "", map[string]string{"addresses": args}, nil
		}
	}
	return entities.ConditionNotConfirmed, "Waiting, All of agents of ETCD role are not ready yet.", nil, nil
}
