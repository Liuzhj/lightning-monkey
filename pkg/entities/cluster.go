package entities

import (
	"github.com/globalsign/mgo/bson"
	"strings"
	"time"
)

type ConditionCheckedResult int
type AgentStatusFlag int

const (
	AgentRole_Master                                      = "master"
	AgentRole_ETCD                                        = "etcd"
	AgentRole_Minion                                      = "minion"
	AgentRole_HA                                          = "ha"
	ClusterNew                                            = "New"
	ClusterProvisioning                                   = "Provisiong"
	ClusterUncontrollable                                 = "Uncontrollable"
	ClusterReady                                          = "Ready"
	ClusterDeleted                                        = "ClusterDeleted"
	ClusterBlockedAgentRegistering                        = "ClusterBlockedAgentRegistering"
	_                              ConditionCheckedResult = iota
	ConditionConfirmed
	ConditionNotConfirmed
	ConditionInapplicable
	MasterSettings_PodCIDR                                    = "pod_ip_cidr"
	MasterSettings_ServiceCIDR                                = "service_ip_cidr"
	MasterSettings_ServiceDNSDomain                           = "service_dns_domain"
	MasterSettings_KubernetesVersion                          = "k8s_version"
	MasterSettings_DockerRegistry                             = "docker_registry"
	MasterSettings_DockerExtraGraphPath                       = "docker_extra_graph_path"
	MasterSettings_MaxPodCountPerNode                         = "max_pod_count"
	MasterSettings_ServiceDNSClusterIP                        = "service_dns_cluster_ip"
	MasterSettings_ExpectedETCDNodeCount                      = "expected_etcd_node_count"
	MasterSettings_APIServerVIP                               = "ha_settings"
	MasterSettings_PortRange                                  = "port_range"
	MasterSettings_ResourceReservation_Kube                   = "kube_res_reserve"
	MasterSettings_ResourceReservation_System                 = "system_res_reserve"
	NetworkStack_Flannel                                      = "flannel"
	NetworkStack_Calico                                       = "calico"
	NetworkStack_KubeRouter                                   = "kuberouter"
	DNS_KubeDNS                                               = "kubedns"
	DNS_CoreDNS                                               = "coredns"
	EXT_DEPLOYMENT_PROMETHEUS                                 = "prometheus"
	EXT_DEPLOYMENT_ALTERMANAGER                               = "altermanager"
	EXT_DEPLOYMENT_METRICSERVER                               = "metric-server"
	EXT_DEPLOYMENT_TRAEFIK                                    = "traefik"
	EXT_DEPLOYMENT_ES                                         = "es"
	EXT_DEPLOYMENT_FILEBEAT                                   = "filebeat"
	EXT_DEPLOYMENT_HELM                                       = "helm"
	EXT_DEPLOYMENT_METRICBEAT                                 = "metricbeat"
	_                                         AgentStatusFlag = iota
	AgentStatusFlag_Whatever
	AgentStatusFlag_Running
	AgentStatusFlag_Provisioned
)

type Cluster struct {
	Id                         *bson.ObjectId        `json:"id" bson:"_id"`
	CreateTime                 time.Time             `json:"create_time" bson:"create_time"`
	HasProvisionedNetworkStack bool                  `json:"has_provisioned_network_stack" bson:"has_provisioned_network_stack"`
	NetworkStackProvisionTime  time.Time             `json:"network_stack_provision_time" bson:"network_stack_provision_time"`
	LastStatusChangeTime       time.Time             `json:"last_status_change_time" bson:"last_status_change_time"`
	Name                       string                `json:"name" bson:"name"`
	ExpectedETCDCount          int                   `json:"expected_etcd_count" bson:"expected_etcd_count"`
	ServiceCIDR                string                `json:"service_cidr" bson:"service_cidr"`
	KubernetesVersion          string                `json:"kubernetes_version" bson:"kubernetes_version"`
	PodNetworkCIDR             string                `json:"pod_network_cidr" bson:"pod_network_cidr"`
	SecurityToken              string                `json:"security_token" bson:"security_token"`
	Status                     string                `json:"status" bson:"status"`
	ServiceDNSDomain           string                `json:"service_dns_domain" bson:"service_dns_domain"`
	NetworkStack               *NetworkStackSettings `json:"network_stack" bson:"network_stack"`
}

type LightningMonkeyClusterSettings struct {
	Id                            string                                      `json:"id"`
	CreateTime                    time.Time                                   `json:"create_time"`
	Name                          string                                      `json:"name"`
	ExpectedETCDCount             int                                         `json:"expected_etcd_count"`
	ServiceCIDR                   string                                      `json:"service_cidr"`
	KubernetesVersion             string                                      `json:"kubernetes_version"`
	PodNetworkCIDR                string                                      `json:"pod_network_cidr"`
	SecurityToken                 string                                      `json:"security_token"`
	ServiceDNSDomain              string                                      `json:"service_dns_domain"`
	ServiceDNSClusterIP           string                                      `json:"service_dns_cluster_ip"`
	MaximumAllowedPodCountPerNode int                                         `json:"maximum_allowed_pod_count_per_node"`
	NetworkStack                  *NetworkStackSettings                       `json:"network_stack"`
	DNSSettings                   *DNSDeploymentSettings                      `json:"dns_deployment_settings"`
	HASettings                    *HASettings                                 `json:"ha_settings"`
	ExtensionalDeployments        map[string] /*key->args*/ map[string]string `json:"ext_deployments"`
	ExtraDockerGraphPath          string                                      `json:"extra_docker_graph_path"`
	PortRangeSettings             *NodePortRangeSettings                      `json:"node_port_range_settings"`
	ResourceReservation           *ResourceReservationSettings                `json:"resource_reservation"`
	HelmSettings                  *HelmSettings                               `json:"helm_settings"`
	ImagePullSecrets              []ImagePullSecret                           `json:"image_pull_secrets"`
}

type ImagePullSecret struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Registry  string `json:"registry"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Email     string `json:"email"`
}

type HelmSettings struct {
	Repositories []HelmRepo  `json:"repositories"`
	Charts       []HelmChart `json:"charts"`
}

type HelmRepo struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type HelmChart struct {
	Name       string               `json:"name"`
	Namespace  string               `json:"namespace"`
	Chart      string               `json:"chart"`
	Version    string               `json:"version"`
	Username   string               `json:"username"`
	Password   string               `json:"password"`
	Repository string               `json:"repository"`
	Parameters []HelmChartParameter `json:"parameters"`
}

type HelmChartParameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceReservationSettings struct {
	Kube   string `json:"kube"`
	System string `json:"system"`
}

type NodePortRangeSettings struct {
	Begin int `json:"begin"`
	End   int `json:"end"`
}

type HASettings struct {
	VIP       string `json:"vip"`
	RouterID  string `json:"router_id"`
	NodeCount int    `json:"count"`
}

type ClusterStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type NetworkStackSettings struct {
	Type       string            `json:"type"` //flannel, kube-router, calico...
	Attributes map[string]string `json:"attributes" bson:"attributes"`
}

type DNSDeploymentSettings struct {
	Type       string            `json:"type"` //kubedns,coredns...
	Attributes map[string]string `json:"attributes" bson:"attributes"`
}

type Certificate struct {
	Id         *bson.ObjectId `json:"id" bson:"_id"`
	ClusterId  *bson.ObjectId `json:"cluster_id" bson:"cluster_id"`
	Name       string         `json:"name" bson:"name"`
	Content    string         `json:"content" bson:"content"`
	CreateTime time.Time      `json:"create_time" bson:"create_time"`
}

type CertificateCollection []*Certificate

func (c CertificateCollection) GetCertificateContent(name string) string {
	for i := 0; i < len(c); i++ {
		if strings.ToLower(c[i].Name) == strings.ToLower(name) {
			return c[i].Content
		}
	}
	return ""
}

type CertificateKeyPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type LightningMonkeyCertificateCollection []*CertificateKeyPair

func (c LightningMonkeyCertificateCollection) GetCertificateContent(name string) string {
	for i := 0; i < len(c); i++ {
		if strings.ToLower(c[i].Name) == strings.ToLower(name) {
			return c[i].Value
		}
	}
	return ""
}
