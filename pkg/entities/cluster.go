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
	Id                   *bson.ObjectId `json:"id" bson:"_id"`
	CreateTime           time.Time      `json:"create_time" bson:"create_time"`
	LastStatusChangeTime time.Time      `json:"last_status_change_time" bson:"last_status_change_time"`
	Name                 string         `json:"name" bson:"name"`
	ExpectedETCDCount    int            `json:"expected_etcd_count" bson:"expected_etcd_count"`
	ServiceCIDR          string         `json:"service_cidr" bson:"service_cidr"`
	KubernetesVersion    string         `json:"kubernetes_version" bson:"kubernetes_version"`
	PodNetworkCIDR       string         `json:"pod_network_cidr" bson:"pod_network_cidr"`
	SecurityToken        string         `json:"security_token" bson:"security_token"`
	Status               string         `json:"status" bson:"status"`
}

type Certificate struct {
	Id         *bson.ObjectId `json:"id" bson:"_id"`
	ClusterId  *bson.ObjectId `json:"cluster_id" bson:"cluster_id"`
	Name       string         `json:"name" bson:"name"`
	Content    string         `json:"content" bson:"content"`
	CreateTime time.Time      `json:"create_time" bson:"create_time"`
}

type Agent struct {
	Id             *bson.ObjectId `json:"id" bson:"_id"`
	ClusterId      *bson.ObjectId `json:"cluster_id" bson:"cluster_id"`
	Hostname       string         `json:"hostname" bson:"hostname"`
	IP             string         `json:"ip" bson:"ip"`
	LastReportTime time.Time      `json:"last_report_time" bson:"last_report_time"`
	Role           string         `json:"role" bson:"role"`
}
