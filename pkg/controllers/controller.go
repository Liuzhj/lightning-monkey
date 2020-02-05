package controllers

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
)

type DeploymentController interface {
	Initialize(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) error
	Install() error
	UnInstall() error
	GetName() string
	HasInstalled() (bool, error)
}
