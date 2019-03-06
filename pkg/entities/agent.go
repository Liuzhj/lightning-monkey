package entities

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

const (
	AgentJob_Deploy_Master   = "MASTER"
	AgentJob_Deploy_ETCD     = "ETCD"
	AgentJob_Deploy_Minion   = "Minion"
	AgentStatus_Registered   = "New"
	AgentStatus_Running      = "Running"
	AgentStatus_Provisioning = "Provisioning"
)

type Agent struct {
	Id                             *bson.ObjectId `json:"id" bson:"_id"`
	ClusterId                      *bson.ObjectId `json:"cluster_id" bson:"cluster_id"`
	MetadataId                     string         `json:"metadata_id" bson:"metadata_id"`
	Hostname                       string         `json:"hostname" bson:"hostname"`
	LastReportIP                   string         `json:"last_report_ip" bson:"last_report_ip"`
	LastReportStatus               string         `json:"last_report_status" bson:"last_report_status"`
	LastReportTime                 time.Time      `json:"last_report_time" bson:"last_report_time"`
	Reason                         string         `json:"reason" bson:"reason"`
	HasProvisionedMasterComponents bool           `json:"provisioned_master_components" bson:"provisioned_master_components"`
	MasterComponentsProvisionTime  time.Time      `json:"master_components_provision_time" bson:"master_components_provision_time"`
	HasProvisionedETCD             bool           `json:"provisioned_etcd" bson:"provisioned_etcd"`
	ETCDProvisionTime              time.Time      `json:"etcd_provision_time" bson:"etcd_provision_time"`
	HasProvisionedMinion           bool           `json:"provisioned_minion" bson:"provisioned_minion"`
	MinionProvisionTime            time.Time      `json:"minion_provision_time" bson:"minion_provision_time"`
	IsDelete                       bool           `json:"is_delete" bson:"is_delete"`
	HasETCDRole                    bool           `json:"has_etcd_role" bson:"has_etcd_role"`
	HasMasterRole                  bool           `json:"has_master_role" bson:"has_master_role"`
	HasMinionRole                  bool           `json:"has_minion_role" bson:"has_minion_role"`
}

type AgentJob struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
	Reason    string            `json:"reason"`
}

type AgentStatus struct {
	IP     string `json:"-"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}
