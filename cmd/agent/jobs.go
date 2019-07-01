package main

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"time"
)

type AgentJobHandler func(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error)
type AgentJobHandlerFactory struct {
	handlers map[string][]AgentJobHandler
}

func (hf *AgentJobHandlerFactory) GetHandler(jobName string) []AgentJobHandler {
	return hf.handlers[jobName]
}

func (hf *AgentJobHandlerFactory) Initialize(c chan<- LightningMonkeyAgentReportStatus, ma *LightningMonkeyAgent) {
	if hf.handlers == nil {
		hf.handlers = map[string][]AgentJobHandler{}
	}
	hf.handlers[entities.AgentJob_Deploy_ETCD] = []AgentJobHandler{HandleDeployETCD, CheckETCDHealth}
	hf.handlers[entities.AgentJob_Deploy_Master] = []AgentJobHandler{HandleDeployMaster, CheckMasterHealth}
	hf.handlers[entities.AgentJob_Deploy_Minion] = []AgentJobHandler{HandleDeployMinion, CheckMinionHealth}
	//initialize health check go-routines.
	for k, v := range hf.handlers {
		go hf.healthCheck(c, ma, k, v[1])
	}
}

//do health check for each of supported Lightning Monkey components.
func (hf *AgentJobHandlerFactory) healthCheck(c chan<- LightningMonkeyAgentReportStatus, ma *LightningMonkeyAgent, key string, hc AgentJobHandler) {
	for {
		succeed, err := hc(nil, ma)
		s := LightningMonkeyAgentReportStatus{
			Key: key,
			Item: entities.LightningMonkeyAgentReportStatusItem{
				HasProvisioned: succeed,
				LastSeenTime:   time.Now(),
			},
		}
		if err != nil {
			s.Item.Reason = err.Error()
		}
		c <- s
		time.Sleep(time.Second * 3)
	}
}
