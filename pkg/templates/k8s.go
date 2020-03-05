package templates

import "github.com/g0194776/lightningmonkey/pkg/entities"

func InitializeKubernetesResourceTemplates() {
	//register ETCD resource template for dynamic YAML configuration file generation.
	SetTemplate(entities.AgentJob_Deploy_ETCD, []string{"1.12", "1.13"}, `apiVersion: "kubeadm.k8s.io/v1alpha3"
kind: ClusterConfiguration
etcd:
    local:
        image: {{.IMAGE}}
        dataDir: {{.DATADIR}}
        serverCertSANs:
        - "{{.HOST}}"
        - "127.0.0.1"
        peerCertSANs:
        - "{{.HOST}}"
        - "127.0.0.1"
        extraArgs:
            initial-cluster: {{.SERVERS}}
            initial-cluster-state: new
            name: {{.NAME}}
            listen-peer-urls: https://{{.ADDR}}:2380
            listen-client-urls: https://{{.ADDR}}:2379
            advertise-client-urls: https://{{.ADDR}}:2379
            initial-advertise-peer-urls: https://{{.ADDR}}:2380`)
	SetTemplate(entities.AgentJob_Deploy_ETCD, []string{"1.14", "1.15", "1.16"}, `apiVersion: "kubeadm.k8s.io/v1beta1"
kind: ClusterConfiguration
etcd:
    local:
        imageTag: {{.IMAGETAG}}
        imageRepository: {{.IMAGEREPO}}
        dataDir: {{.DATADIR}}
        serverCertSANs:
        - "{{.HOST}}"
        - "127.0.0.1"
        peerCertSANs:
        - "{{.HOST}}"
        - "127.0.0.1"
        extraArgs:
            initial-cluster: {{.SERVERS}}
            initial-cluster-state: new
            name: {{.NAME}}
            listen-peer-urls: https://{{.ADDR}}:2380
            listen-client-urls: https://{{.ADDR}}:2379
            advertise-client-urls: https://{{.ADDR}}:2379
            initial-advertise-peer-urls: https://{{.ADDR}}:2380`)
}
