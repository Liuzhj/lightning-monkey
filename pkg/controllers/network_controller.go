package controllers

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	k8s "k8s.io/client-go/kubernetes"
)

func CreateNetworkStackController(client *k8s.Clientset, clientIp string, settings entities.LightningMonkeyClusterSettings) (DeploymentController, error) {
	if settings.NetworkStack == nil {
		return nil, fmt.Errorf("Kubernetes cluster network settings is empty, cluster: %s", settings.Id)
	}
	switch settings.NetworkStack.Type {
	case entities.NetworkStack_KubeRouter:
		c := &KubeRouterNetworkController{}
		return c, c.Initialize(client, clientIp, settings)
	default:
		return nil, fmt.Errorf("No any types of supported network stack were matched with current cluster settings: %s", settings.NetworkStack.Type)
	}
}
