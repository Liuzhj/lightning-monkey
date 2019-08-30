package cache

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"sort"
	"sync"
)

type agentExp func(a *entities.LightningMonkeyAgent) bool
type AgentCache struct {
	*sync.Mutex
	etcd      map[string]*entities.LightningMonkeyAgent
	k8sMaster map[string]*entities.LightningMonkeyAgent
	k8sMinion map[string]*entities.LightningMonkeyAgent
	ha        map[string]*entities.LightningMonkeyAgent
}

func (ac *AgentCache) Initialize() {
	if ac.Mutex == nil {
		ac.Mutex = &sync.Mutex{}
	}
	ac.etcd = make(map[string]*entities.LightningMonkeyAgent)
	ac.k8sMaster = make(map[string]*entities.LightningMonkeyAgent)
	ac.k8sMinion = make(map[string]*entities.LightningMonkeyAgent)
	ac.ha = make(map[string]*entities.LightningMonkeyAgent)
}

func (ac *AgentCache) InitializeWithValues(etcd, k8sMaster, k8sMinion, ha map[string]*entities.LightningMonkeyAgent) {
	if ac.Mutex == nil {
		ac.Mutex = &sync.Mutex{}
	}
	ac.etcd = etcd
	ac.k8sMaster = k8sMaster
	ac.k8sMinion = k8sMinion
	ac.ha = ha
}

func (ac *AgentCache) Online(agent entities.LightningMonkeyAgent) {
	ac.Lock()
	if agent.HasETCDRole {
		ac.etcd[agent.Id] = &agent
	}
	if agent.HasMasterRole {
		ac.k8sMaster[agent.Id] = &agent
	}
	if agent.HasMinionRole {
		ac.k8sMinion[agent.Id] = &agent
	}
	if agent.HasHARole {
		ac.ha[agent.Id] = &agent
	}
	ac.Unlock()
	logrus.Infof("Agent %s online..., etcd-role: %t, master-role: %t, minion-role: %t, ha-role: %t", agent.Id, agent.HasETCDRole, agent.HasMasterRole, agent.HasMinionRole, agent.HasHARole)
}

func (ac *AgentCache) Offline(agent entities.LightningMonkeyAgent) {
	ac.Lock()
	if agent.HasETCDRole {
		delete(ac.etcd, agent.Id)
	}
	if agent.HasMasterRole {
		delete(ac.k8sMaster, agent.Id)
	}
	if agent.HasMinionRole {
		delete(ac.k8sMinion, agent.Id)
	}
	if agent.HasHARole {
		delete(ac.ha, agent.Id)
	}
	ac.Unlock()
	logrus.Infof("Agent %s offline..., etcd-role: %t, master-role: %t, minion-role: %t, ha-role: %t", agent.Id, agent.HasETCDRole, agent.HasMasterRole, agent.HasMinionRole, agent.HasHARole)
}

func (ac *AgentCache) GetTotalCountByRole(role string) int {
	ac.Lock()
	defer ac.Unlock()
	switch role {
	case entities.AgentRole_ETCD:
		return len(ac.etcd)
	case entities.AgentRole_Master:
		return len(ac.k8sMaster)
	case entities.AgentRole_Minion:
		return len(ac.k8sMinion)
	case entities.AgentRole_HA:
		return len(ac.ha)
	default:
		return -1
	}
}

func (ac *AgentCache) GetTotalProvisionedCountByRole(role string) int {
	ac.Lock()
	defer ac.Unlock()
	var f func(a *entities.LightningMonkeyAgent) bool
	var m map[string]*entities.LightningMonkeyAgent
	switch role {
	case entities.AgentRole_ETCD:
		m = ac.etcd
		f = func(a *entities.LightningMonkeyAgent) bool {
			return a.State != nil && a.State.HasProvisionedETCD
		}
	case entities.AgentRole_Master:
		m = ac.k8sMaster
		f = func(a *entities.LightningMonkeyAgent) bool {
			return a.State != nil && a.State.HasProvisionedMasterComponents
		}
	case entities.AgentRole_Minion:
		m = ac.k8sMinion
		f = func(a *entities.LightningMonkeyAgent) bool {
			return a.State != nil && a.State.HasProvisionedMinion
		}
	case entities.AgentRole_HA:
		m = ac.ha
		f = func(a *entities.LightningMonkeyAgent) bool {
			return a.State != nil && a.State.HasProvisionedHA
		}
	default:
		return -1
	}
	cnt := 0
	for _, v := range m {
		if f(v) {
			cnt++
		}
	}
	return cnt
}

func (ac *AgentCache) GetAgentsAddress(role string, mustStatusFlag entities.AgentStatusFlag) []string {
	ac.Lock()
	defer ac.Unlock()
	var expFunc agentExp
	var targetCollection map[string]*entities.LightningMonkeyAgent
	switch role {
	case entities.AgentRole_ETCD:
		targetCollection = ac.etcd
		expFunc = func(a *entities.LightningMonkeyAgent) bool {
			if !a.HasETCDRole {
				return false
			}
			if mustStatusFlag == entities.AgentStatusFlag_Running /*running*/ && !a.IsRunning() {
				return false
			}
			if mustStatusFlag == entities.AgentStatusFlag_Provisioned /*provisioned*/ && (!a.IsRunning() || !a.State.HasProvisionedETCD) {
				return false
			}
			return true
		}
	case entities.AgentRole_Master:
		targetCollection = ac.k8sMaster
		expFunc = func(a *entities.LightningMonkeyAgent) bool {
			if !a.HasMasterRole {
				return false
			}
			if mustStatusFlag == entities.AgentStatusFlag_Running /*running*/ && !a.IsRunning() {
				return false
			}
			if mustStatusFlag == entities.AgentStatusFlag_Provisioned /*provisioned*/ && (!a.IsRunning() || !a.State.HasProvisionedMasterComponents) {
				return false
			}
			return true
		}
	case entities.AgentRole_Minion:
		targetCollection = ac.k8sMinion
		expFunc = func(a *entities.LightningMonkeyAgent) bool {
			if !a.HasMinionRole {
				return false
			}
			if mustStatusFlag == entities.AgentStatusFlag_Running /*running*/ && !a.IsRunning() {
				return false
			}
			return true
		}
	case entities.AgentRole_HA:
		targetCollection = ac.ha
		expFunc = func(a *entities.LightningMonkeyAgent) bool {
			if !a.HasHARole {
				return false
			}
			if mustStatusFlag == entities.AgentStatusFlag_Running /*running*/ && !a.IsRunning() {
				return false
			}
			if mustStatusFlag == entities.AgentStatusFlag_Provisioned /*provisioned*/ && (!a.IsRunning() || !a.State.HasProvisionedHA) {
				return false
			}
			return true
		}
	default:
		//fast fail when occurs internal serious BUG.
		logrus.Fatalf("Illegal type of role name: %s", role)
		return nil
	}
	ips := []string{}
	for _, as := range targetCollection {
		if expFunc(as) {
			ips = append(ips, as.State.LastReportIP)
		}
	}
	sort.Strings(ips)
	return ips
}

func (ac *AgentCache) GetETCDCount() int {
	ac.Lock()
	defer ac.Unlock()
	return len(ac.etcd)
}

func (ac *AgentCache) GetKubernetesMasterCount() int {
	ac.Lock()
	defer ac.Unlock()
	return len(ac.k8sMaster)
}

func (ac *AgentCache) GetKubernetesMinionCount() int {
	ac.Lock()
	defer ac.Unlock()
	return len(ac.k8sMinion)
}

func (ac *AgentCache) GetHACount() int {
	ac.Lock()
	defer ac.Unlock()
	return len(ac.ha)
}

func (ac *AgentCache) GetFirstProvisionedKubernetesMasterAgent() *entities.LightningMonkeyAgent {
	ac.Lock()
	defer ac.Unlock()
	f := func(a *entities.LightningMonkeyAgent) bool {
		return a.State != nil && a.State.HasProvisionedMasterComponents
	}
	for _, v := range ac.k8sMaster {
		if f(v) {
			//make a memory copy.
			agent := *v
			return &agent
		}
	}
	return nil
}
