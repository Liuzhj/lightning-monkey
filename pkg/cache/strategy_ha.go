package cache

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"sort"
	"strconv"
	"strings"
)

type HAJobStrategy struct {
}

func (js *HAJobStrategy) GetStrategyName() string {
	return "HA"
}

func (js *HAJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	if cc.GetSettings().HASettings == nil {
		return entities.ConditionInapplicable, "", nil, nil
	}
	haIps := cache.GetAgentsAddress(entities.AgentRole_HA, entities.AgentStatusFlag_Whatever)
	if haIps == nil || len(haIps) < cc.GetSettings().HASettings.NodeCount {
		return entities.ConditionNotConfirmed, "HAProxy & KeepAlived deployments are postponed, Waiting for enough nodes status to online...", nil, nil
	}
	if agent.HasHARole && !agent.State.HasProvisionedHA {
		routerId := "40"
		masterIps := cache.GetAgentsAddress(entities.AgentRole_Master, entities.AgentStatusFlag_Whatever)
		if cc.GetSettings().HASettings.RouterID != "" {
			routerId = cc.GetSettings().HASettings.RouterID
		}
		//sort before using it to calculate the index.
		sort.Strings(haIps)
		return entities.ConditionConfirmed, "", map[string]string{
			"master-addresses": strings.Join(masterIps, ","),
			"ha-addresses":     strings.Join(haIps, ","),
			"state":            "BACKUP",
			"router-id":        routerId,
			"priority":         strconv.Itoa(js.indexOf(agent.State.LastReportIP, haIps) + 100), //dynamic priority calculation.
			"vip":              cc.GetSettings().HASettings.VIP,
		}, nil
	}
	if cache.GetTotalProvisionedCountByRole(entities.AgentRole_HA) == 0 {
		return entities.ConditionNotConfirmed, "Waiting, Not equals required minimum count of HAProxy & KeepAlived nodes.", nil, nil
	}
	return entities.ConditionInapplicable, "", nil, nil
}

func (js *HAJobStrategy) indexOf(word string, data []string) int {
	for k, v := range data {
		if word == v {
			return k
		}
	}
	return -1
}
