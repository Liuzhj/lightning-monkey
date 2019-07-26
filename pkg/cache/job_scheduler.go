package cache

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
)

type ClusterJobScheduler interface {
	InitializeStrategies()
	GetNextJob(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.AgentJob, error)
}

type ClusterJobSchedulerImple struct {
	strategies []ClusterJobStrategy
}

func (js *ClusterJobSchedulerImple) InitializeStrategies() {
	js.strategies = []ClusterJobStrategy{
		&ClusterETCDJobStrategy{},
		&ClusterKubernetesMasterJobStrategy{},
		&ClusterKubernetesMinionJobStrategy{},
		&ClusterKubernetesNetworkStackJobStrategy{},
	}
}

func (js *ClusterJobSchedulerImple) GetNextJob(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.AgentJob, error) {
	if js.strategies == nil || len(js.strategies) == 0 {
		return entities.AgentJob{Name: entities.AgentJob_NOP, Reason: "Skipped, No any cluster job strategies being found."}, nil
	}
	if agent.State == nil {
		return entities.AgentJob{Name: entities.AgentJob_NOP, Reason: "Occurred internal exceptions!"}, fmt.Errorf("Current agent: %s state is not online yet!", agent.Id)
	}
	var deployFlag entities.ConditionCheckedResult
	var deployArgs map[string]string
	var reason string
	var err error
	for i := 0; i < len(js.strategies); i++ {
		deployFlag, reason, deployArgs, err = js.strategies[i].CanDeploy(cc, agent, cache)
		if err != nil {
			return entities.AgentJob{Name: entities.AgentJob_NOP, Reason: "Occurred internal error!"}, err
		}
		if deployFlag == entities.ConditionInapplicable {
			continue
		}
		if deployFlag == entities.ConditionNotConfirmed {
			return entities.AgentJob{Name: entities.AgentJob_NOP, Reason: reason}, nil
		}
		return entities.AgentJob{Name: js.strategies[i].GetStrategyName(), Arguments: deployArgs}, nil
	}
	return entities.AgentJob{Name: entities.AgentJob_NOP, Reason: "Waiting, no any operations should perform."}, nil
}
