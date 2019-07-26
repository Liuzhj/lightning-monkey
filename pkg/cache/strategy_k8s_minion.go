package cache

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"strings"
)

type ClusterKubernetesMinionJobStrategy struct {
}

func (js *ClusterKubernetesMinionJobStrategy) GetStrategyName() string {
	return entities.AgentJob_Deploy_Minion
}

func (js *ClusterKubernetesMinionJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	if agent.HasMinionRole && !agent.State.HasProvisionedMinion {
		masterIps := cache.GetAgentsAddress(entities.AgentRole_Master, entities.AgentStatusFlag_Provisioned)
		return entities.ConditionConfirmed, "", map[string]string{"addresses": strings.Join(masterIps, ",")}, nil
	}
	return entities.ConditionInapplicable, "", nil, nil
}
