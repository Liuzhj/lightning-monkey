package main

import "github.com/g0194776/lightningmonkey/pkg/entities"

type AgentJobHandler func(job *entities.AgentJob) error
type AgentJobHandlerFactory struct {
	handlers map[string]AgentJobHandler
}

func (hf *AgentJobHandlerFactory) Register(handler AgentJobHandler) {}

func (hf *AgentJobHandlerFactory) GetHandler(jobName string) AgentJobHandler {
	return nil
}

func (hf *AgentJobHandlerFactory) Initialize() {}
