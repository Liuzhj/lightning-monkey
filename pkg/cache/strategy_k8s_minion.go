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
		vip := ""
		masterIps := cache.GetAgentsAddress(entities.AgentRole_Master, entities.AgentStatusFlag_Provisioned)
		if cc.GetSettings().HASettings != nil {
			vip = cc.GetSettings().HASettings.VIP
		}
		return entities.ConditionConfirmed, "", map[string]string{
			"addresses":  strings.Join(masterIps, ","),
			"ha_address": vip,
		}, nil
	}
	return entities.ConditionInapplicable, "", nil, nil
}
