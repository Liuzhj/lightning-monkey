package network

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/controllers/network/router"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
)

func CreateNetworkStackController(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) (*router.KubeRouterNetworkController, error) {
	if settings.NetworkStack == nil {
		return nil, fmt.Errorf("Kubernetes cluster network settings is empty, cluster: %s", settings.Id)
	}
	switch settings.NetworkStack.Type {
	case entities.NetworkStack_KubeRouter:
		c := &router.KubeRouterNetworkController{}
		return c, c.Initialize(client, clientIp, settings)
	default:
		return nil, fmt.Errorf("No any types of supported network stack were matched with current cluster settings: %s", settings.NetworkStack.Type)
	}
}
