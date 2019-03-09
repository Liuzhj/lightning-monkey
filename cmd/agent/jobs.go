package main

import "github.com/g0194776/lightningmonkey/pkg/entities"

type AgentJobHandler func(job *entities.AgentJob, arg *AgentArgs) error
type AgentJobHandlerFactory struct {
	handlers map[string]AgentJobHandler
}

func (hf *AgentJobHandlerFactory) GetHandler(jobName string) AgentJobHandler {
	return hf.handlers[jobName]
}

func (hf *AgentJobHandlerFactory) Initialize() {
	if hf.handlers == nil {
		hf.handlers = map[string]AgentJobHandler{}
	}
	hf.handlers[entities.AgentJob_Deploy_ETCD] = HandleDeployETCD
}
