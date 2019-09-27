package main

import (
	"context"
	"github.com/docker/engine-api/types"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"strings"
)

func HandleDeployMinion(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	if job.Arguments == nil || (job.Arguments["addresses"] == "" && job.Arguments["ha_address"] == "") {
		return false, xerrors.Errorf("Illegal Minion deployment job, required arguments are missed %w", crashError)
	}
	var masterIP string
	//use VIP to communicate with Kubernetes Master is the top priority.
	if job.Arguments["ha_address"] != "" {
		masterIP = job.Arguments["ha_address"]
	} else {
		//instead, pick one of Kubernetes master address is not HA solution.
		masterIP = strings.Split(job.Arguments["addresses"], ",")[0]
	}
	err := a.runKubeletContainer(masterIP)
	return err == nil, err
}

func CheckMinionHealth(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	var err error
	var containers []types.Container
	containers, err = a.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		logrus.Errorf("Failed to retrieve all containers information, error: %s", err.Error())
		return false, err
	}
	if containers == nil || len(containers) == 0 {
		return false, nil
	}
	for i := 0; i < len(containers); i++ {
		logrus.Debugf("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if containers[i].Names[0] == "/kubelet" &&
			strings.Contains(containers[i].Status, "Up") {
			return *a.arg.IsMinionRole, nil //considered with role assignment, minion provision state will not directly return.
		}
	}
	return false, nil
}
