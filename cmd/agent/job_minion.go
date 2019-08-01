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
	if job.Arguments == nil || job.Arguments["addresses"] == "" {
		return false, xerrors.Errorf("Illegal Minion deployment job, required arguments are missed %w", crashError)
	}
	servers := strings.Split(job.Arguments["addresses"], ",")
	err := a.runKubeletContainer(servers[0])
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
		logrus.Infof("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if containers[i].Names[0] == "/kubelet" &&
			strings.Contains(containers[i].Status, "Up") {
			return *a.arg.IsMinionRole, nil //considered with role assignment, minion provision state will not directly return.
		}
	}
	return false, nil
}
