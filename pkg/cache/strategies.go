package cache

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
)

type ClusterJobStrategy interface {
	GetStrategyName() string
	CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error)
}
