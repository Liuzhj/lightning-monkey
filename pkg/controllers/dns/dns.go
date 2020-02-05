package dns

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/controllers/dns/coredns"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
)

func CreateDNSDeploymentController(cs *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) (*coredns.CoreDNSController, error) {
	if settings.DNSSettings == nil {
		return nil, fmt.Errorf("Kubernetes cluster DNS deployment settings is empty, cluster: %s", settings.Id)
	}
	switch settings.DNSSettings.Type {
	case entities.DNS_CoreDNS:
		c := &coredns.CoreDNSController{}
		return c, c.Initialize(cs, clientIp, settings)
	default:
		return nil, fmt.Errorf("No any types of supported DNS deployment strategy were matched with current cluster settings: %s", settings.NetworkStack.Type)
	}
}
