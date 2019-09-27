package cache

import "github.com/g0194776/lightningmonkey/pkg/entities"

type ClusterKubernetesMasterJobStrategy struct {
}

func (js *ClusterKubernetesMasterJobStrategy) GetStrategyName() string {
	return entities.AgentJob_Deploy_Master
}

func (js *ClusterKubernetesMasterJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	if agent.HasMasterRole && !agent.State.HasProvisionedMasterComponents {
		return entities.ConditionConfirmed, "", nil, nil
	}
	if cache.GetTotalProvisionedCountByRole(entities.AgentRole_Master) <= 0 {
		return entities.ConditionNotConfirmed, "Waiting, All of agents of Kubernetes master role are not ready yet.", nil, nil
	}
	return entities.ConditionInapplicable, "", nil, nil
}
