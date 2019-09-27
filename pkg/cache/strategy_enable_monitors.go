package cache

import "github.com/g0194776/lightningmonkey/pkg/entities"

type EnableMonitorsJobStrategy struct {
}

func (js *EnableMonitorsJobStrategy) GetStrategyName() string {
	return "Enable Cluster Monitors"
}

func (js *EnableMonitorsJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	cc.EnableMonitors()
	return entities.ConditionInapplicable, "", nil, nil
}
