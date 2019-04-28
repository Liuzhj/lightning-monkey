package controllers

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	k8s "k8s.io/client-go/kubernetes"
)

type NetworkStackController interface {
	Initialize(client *k8s.Clientset, cluster *entities.Cluster) error
	Install() error
	UnInstall() error
}

func CreateNetworkStackController(client *k8s.Clientset, cluster *entities.Cluster) (NetworkStackController, error) {
	if cluster.NetworkStack == nil {
		return nil, fmt.Errorf("Kubernetes cluster network settings is empty, cluster: %s", cluster.Id.Hex())
	}
	switch cluster.NetworkStack.Type {
	case entities.NetworkStack_KubeRouter:
		c := &KubeRouterNetworkController{}
		return c, c.Initialize(client, cluster)
	default:
		return nil, fmt.Errorf("No any types of supported network stack were matched current cluster settings: %s", cluster.NetworkStack.Type)
	}
}
