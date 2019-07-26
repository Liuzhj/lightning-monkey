package controllers

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	k8s "k8s.io/client-go/kubernetes"
)

type NetworkStackController interface {
	Initialize(client *k8s.Clientset, settings entities.LightningMonkeyClusterSettings) error
	Install() error
	UnInstall() error
	GetName() string
	HasInstalled() (bool, error)
}

func CreateNetworkStackController(client *k8s.Clientset, settings entities.LightningMonkeyClusterSettings) (NetworkStackController, error) {
	if settings.NetworkStack == nil {
		return nil, fmt.Errorf("Kubernetes cluster network settings is empty, cluster: %s", settings.Id)
	}
	switch settings.NetworkStack.Type {
	case entities.NetworkStack_KubeRouter:
		c := &KubeRouterNetworkController{}
		return c, c.Initialize(client, settings)
	default:
		return nil, fmt.Errorf("No any types of supported network stack were matched current cluster settings: %s", settings.NetworkStack.Type)
	}
}
