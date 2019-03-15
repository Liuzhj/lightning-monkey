package main

import "github.com/g0194776/lightningmonkey/pkg/entities"

func HandleDeployMaster(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	return false, nil
}

func CheckMasterHealth(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	return false, nil
}
