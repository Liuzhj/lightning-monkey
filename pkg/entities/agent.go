package entities

import (
	"github.com/globalsign/mgo/bson"
	"time"
)

type Agent struct {
	Id                             *bson.ObjectId `json:"id" bson:"_id"`
	ClusterId                      *bson.ObjectId `json:"cluster_id" bson:"cluster_id"`
	MetadataId                     string         `json:"metadata_id" bson:"metadata_id"`
	Hostname                       string         `json:"hostname" bson:"hostname"`
	LastReportIP                   string         `json:"last_report_ip" bson:"last_report_ip"`
	LastReportStatus               string         `json:"last_report_status" bson:"last_report_status"`
	LastReportTime                 time.Time      `json:"last_report_time" bson:"last_report_time"`
	Roles                          []string       `json:"roles" bson:"roles"`
	HasProvisionedMasterComponents bool           `json:"provisioned_master_components" bson:"provisioned_master_components"`
	MasterComponentsProvisionTime  time.Time      `json:"master_components_provision_time" bson:"master_components_provision_time"`
	HasProvisionedETCD             bool           `json:"provisioned_etcd" bson:"provisioned_etcd"`
	ETCDProvisionTime              time.Time      `json:"etcd_provision_time" bson:"etcd_provision_time"`
	HasProvisionedMinion           bool           `json:"provisioned_minion" bson:"provisioned_minion"`
	MinionProvisionTime            time.Time      `json:"minion_provision_time" bson:"minion_provision_time"`
	IsDelete                       bool           `json:"is_delete" bson:"is_delete"`
}
