package main

import (
	"context"
	"github.com/docker/engine-api/types"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"strings"
)

func HandleDeployMaster(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	err := common.CertManager.GenerateMasterCertificatesAndManifest(
		a.masterSettings[entities.MasterSettings_KubernetesVersion],
		CERTIFICATE_STORAGE_PATH,
		*a.arg.Address,
		a.masterSettings,
		a.basicImages,
	)
	if err != nil {
		return false, xerrors.Errorf("Failed to generate Kubernetes master certificates and manifests, error: %s %w", err.Error(), crashError)
	}
	return true, nil
}

func CheckMasterHealth(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
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
	return (hasAPIServerStarted(containers) && hasControllerManagerStarted(containers) && hasSchedulerStarted(containers)), nil
}

func hasAPIServerStarted(containers []types.Container) bool {
	for i := 0; i < len(containers); i++ {
		logrus.Infof("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if strings.Contains(containers[i].Names[0], "k8s_kube-apiserver") &&
			strings.Contains(containers[i].Names[0], "kube-system") &&
			strings.Contains(containers[i].Status, "Up") {
			return true
		}
	}
	return false
}

func hasControllerManagerStarted(containers []types.Container) bool {
	for i := 0; i < len(containers); i++ {
		logrus.Infof("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if strings.Contains(containers[i].Names[0], "k8s_kube-controller-manager") &&
			strings.Contains(containers[i].Names[0], "kube-system") &&
			strings.Contains(containers[i].Status, "Up") {
			return true
		}
	}
	return false
}

func hasSchedulerStarted(containers []types.Container) bool {
	for i := 0; i < len(containers); i++ {
		logrus.Infof("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if strings.Contains(containers[i].Names[0], "k8s_kube-scheduler") &&
			strings.Contains(containers[i].Names[0], "kube-system") &&
			strings.Contains(containers[i].Status, "Up") {
			return true
		}
	}
	return false
}
