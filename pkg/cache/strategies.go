package cache

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"math"
	"strings"
)

type ClusterJobStrategy interface {
	GetStrategyName() string
	CanDeploy(clusterSettings entities.LightningMonkeyClusterSettings, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error)
}

type ClusterETCDJobStrategy struct {
}

func (js *ClusterETCDJobStrategy) GetStrategyName() string {
	return entities.AgentJob_Deploy_ETCD
}

func (js *ClusterETCDJobStrategy) CanDeploy(clusterSettings entities.LightningMonkeyClusterSettings, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	if agent.HasETCDRole && !agent.State.HasProvisionedETCD {
		return entities.ConditionConfirmed, "", map[string]string{"addresses": strings.Join(cache.GetAgentsAddress(entities.AgentRole_ETCD, entities.AgentStatusFlag_Whatever), ",")}, nil
	}
	if cache.GetTotalProvisionedCountByRole(entities.AgentRole_ETCD) >= 1 && clusterSettings.ExpectedETCDCount == 1 {
		return entities.ConditionInapplicable, "", nil, nil
	}
	if cache.GetTotalProvisionedCountByRole(entities.AgentRole_ETCD) >= int(math.Ceil(float64(float64(clusterSettings.ExpectedETCDCount)/2))) {
		return entities.ConditionInapplicable, "", nil, nil
	}
	return entities.ConditionNotConfirmed, "Waiting, All of agents of ETCD role are not ready yet.", nil, nil
}

type ClusterKubernetesMasterJobStrategy struct {
}

func (js *ClusterKubernetesMasterJobStrategy) GetStrategyName() string {
	return entities.AgentJob_Deploy_Master
}

func (js *ClusterKubernetesMasterJobStrategy) CanDeploy(clusterSettings entities.LightningMonkeyClusterSettings, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	if agent.HasMasterRole && !agent.State.HasProvisionedMasterComponents {
		return entities.ConditionConfirmed, "", nil, nil
	}
	if cache.GetTotalProvisionedCountByRole(entities.AgentRole_Master) <= 0 {
		return entities.ConditionNotConfirmed, "Waiting, All of agents of Kubernetes master role are not ready yet.", nil, nil
	}
	return entities.ConditionInapplicable, "", nil, nil
}

type ClusterKubernetesMinionJobStrategy struct {
}

func (js *ClusterKubernetesMinionJobStrategy) GetStrategyName() string {
	return entities.AgentJob_Deploy_Minion
}

func (js *ClusterKubernetesMinionJobStrategy) CanDeploy(clusterSettings entities.LightningMonkeyClusterSettings, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	if agent.HasMinionRole && !agent.State.HasProvisionedMinion {
		masterIps := cache.GetAgentsAddress(entities.AgentRole_Master, entities.AgentStatusFlag_Provisioned)
		return entities.ConditionConfirmed, "", map[string]string{"addresses": strings.Join(masterIps, ",")}, nil
	}
	return entities.ConditionInapplicable, "", nil, nil
}
