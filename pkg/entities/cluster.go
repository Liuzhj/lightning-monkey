package entities

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type ClusterRole int

const (
	_ ClusterRole = iota
	Master
	ETCD
	Minion
	Master_TCD
	Master_ETCD_Minion
)

type Cluster struct {
	Id                *bson.ObjectId `json:"id"`
	Name              string         `json:"name"`
	ExpectedETCDCount int            `json:"expected_etcd_count"`
	ServiceCIDR       string         `json:"service_cidr"`
	KubernetesVersion string         `json:"kubernetes_version"`
	PodNetworkCIDR    string         `json:"pod_network_cidr"`
	SecurityToken     string         `json:"security_token"`
}

type Certificate struct {
	Id         *bson.ObjectId `json:"id"`
	ClusterId  *bson.ObjectId `json:"cluster_id"`
	Name       string         `json:"name"`
	Content    string         `json:"content"`
	CreateTime time.Time      `json:"create_time"`
}

type Agent struct {
	Id             *bson.ObjectId `json:"id"`
	ClusterId      *bson.ObjectId `json:"cluster_id"`
	Hostname       string         `json:"hostname"`
	IP             string         `json:"ip"`
	LastReportTime time.Time      `json:"last_report_time"`
	Role           string         `json:"role"`
}
