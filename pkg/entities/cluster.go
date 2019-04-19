package entities

import (
	"github.com/globalsign/mgo/bson"
	"strings"
	"time"
)

type ConditionCheckedResult int

const (
	AgentRole_Master                                      = "master"
	AgentRole_ETCD                                        = "etcd"
	AgentRole_Minion                                      = "minion"
	ClusterNew                                            = "New"
	ClusterProvisioning                                   = "Provisiong"
	ClusterReady                                          = "Ready"
	ClusterDeleted                                        = "ClusterDeleted"
	ClusterBlockedAgentRegistering                        = "ClusterBlockedAgentRegistering"
	_                              ConditionCheckedResult = iota
	ConditionConfirmed
	ConditionNotConfirmed
	ConditionInapplicable
	MasterSettings_PodCIDR           = "pod_ip_cidr"
	MasterSettings_ServiceCIDR       = "service_ip_cidr"
	MasterSettings_ServiceDNSDomain  = "service_dns_domain"
	MasterSettings_KubernetesVersion = "k8s_version"
	MasterSettings_DockerRegistry    = "docker_registry"
)

type Cluster struct {
	Id                         *bson.ObjectId `json:"id" bson:"_id"`
	CreateTime                 time.Time      `json:"create_time" bson:"create_time"`
	HasProvisionedNetworkStack bool           `json:"has_provisioned_network_stack" bson:"has_provisioned_network_stack"`
	NetworkStackProvisionTime  time.Time      `json:"network_stack_provision_time" bson:"network_stack_provision_time"`
	LastStatusChangeTime       time.Time      `json:"last_status_change_time" bson:"last_status_change_time"`
	Name                       string         `json:"name" bson:"name"`
	ExpectedETCDCount          int            `json:"expected_etcd_count" bson:"expected_etcd_count"`
	ServiceCIDR                string         `json:"service_cidr" bson:"service_cidr"`
	KubernetesVersion          string         `json:"kubernetes_version" bson:"kubernetes_version"`
	PodNetworkCIDR             string         `json:"pod_network_cidr" bson:"pod_network_cidr"`
	SecurityToken              string         `json:"security_token" bson:"security_token"`
	Status                     string         `json:"status" bson:"status"`
	ServiceDNSDomain           string         `json:"service_dns_domain" bson:"service_dns_domain"`
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
