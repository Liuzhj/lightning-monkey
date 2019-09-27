package controllers

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	k8s "k8s.io/client-go/kubernetes"
)

type DeploymentController interface {
	Initialize(client *k8s.Clientset, clientIp string, settings entities.LightningMonkeyClusterSettings) error
	Install() error
	UnInstall() error
	GetName() string
	HasInstalled() (bool, error)
}

func CreateDNSDeploymentController(client *k8s.Clientset, clientIp string, settings entities.LightningMonkeyClusterSettings) (DeploymentController, error) {
	if settings.DNSSettings == nil {
		return nil, fmt.Errorf("Kubernetes cluster DNS deployment settings is empty, cluster: %s", settings.Id)
	}
	switch settings.DNSSettings.Type {
	case entities.DNS_CoreDNS:
		c := &CoreDNSController{}
		return c, c.Initialize(client, clientIp, settings)
	default:
		return nil, fmt.Errorf("No any types of supported DNS deployment strategy were matched with current cluster settings: %s", settings.NetworkStack.Type)
	}
}
