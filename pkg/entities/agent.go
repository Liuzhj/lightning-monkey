package entities

import (
	"time"
)

const (
	AgentJob_Deploy_Master                  = "MASTER"
	AgentJob_Deploy_ETCD                    = "ETCD"
	AgentJob_Deploy_Minion                  = "Minion"
	AgentJob_Deploy_HA                      = "HA"
	AgentJob_Deploy_NetworkStack_KubeRouter = "Kube-Router"
	AgentJob_NOP                            = "NOP"
	AgentStatus_Registered                  = "New"
	AgentStatus_Running                     = "Running"
	AgentStatus_Provisioning                = "Provisioning"
	AgentStatus_Provision_Succeed           = "Provision Succeed"
	AgentStatus_Provision_Failed            = "Provision Failed"
	AgentReport_Provision                   = "Provision"
	AgentReport_Heartbeat                   = "Heartbeat"
	MaxAgentReportTimeoutSecs               = 30
)

type Agent struct {
	ClusterId                      string    `json:"cluster_id"`
	MetadataId                     string    `json:"metadata_id"`
	Hostname                       string    `json:"hostname"`
	LastReportIP                   string    `json:"last_report_ip"`
	LastReportStatus               string    `json:"last_report_status"`
	LastReportTime                 time.Time `json:"last_report_time"`
	Reason                         string    `json:"reason"`
	HasProvisionedMasterComponents bool      `json:"provisioned_master_components"`
	MasterComponentsProvisionTime  time.Time `json:"master_components_provision_time"`
	HasProvisionedETCD             bool      `json:"provisioned_etcd"`
	ETCDProvisionTime              time.Time `json:"etcd_provision_time"`
	HasProvisionedMinion           bool      `json:"provisioned_minion"`
	MinionProvisionTime            time.Time `json:"minion_provision_time"`
	IsDelete                       bool      `json:"is_delete"`
	HasETCDRole                    bool      `json:"has_etcd_role"`
	HasMasterRole                  bool      `json:"has_master_role"`
	HasMinionRole                  bool      `json:"has_minion_role"`
	HasHARole                      bool      `json:"has_ha_role"`
	ListenPort                     int       `json:"listen_port"`
}

type LightningMonkeyAgent struct {
	Id               string          `json:"id" bson:"_id"`
	ClusterId        string          `json:"cluster_id" bson:"cluster_id"`
	AdminCertificate string          `json:"admin_certificate"` //not exist if it has not master role.
	Hostname         string          `json:"hostname" bson:"hostname"`
	IsDelete         bool            `json:"is_delete" bson:"is_delete"`
	HasETCDRole      bool            `json:"has_etcd_role" bson:"has_etcd_role"`
	HasMasterRole    bool            `json:"has_master_role" bson:"has_master_role"`
	HasMinionRole    bool            `json:"has_minion_role" bson:"has_minion_role"`
	HasHARole        bool            `json:"has_ha_role"`
	HostInformation  HostInformation `json:"host_information"`
	ListenPort       int             `json:"listen_port"`
	State            *AgentState     `json:"-"`
}

type HostInformation struct {
	OS            string  `json:"os"`
	Kernel        string  `json:"kernel"`
	CPUCores      int32   `json:"cpu_cores"`
	CPUMhz        float64 `json:"cpu_mhz"`
	MemoryTotalMB uint64  `json:"memory_total_mb"`
}

type AgentState struct {
	LastReportIP                   string    `json:"last_report_ip"`
	LastReportTime                 time.Time `json:"last_report_time"`
	Reason                         string    `json:"reason"`
	HasProvisionedMasterComponents bool      `json:"provisioned_master_components"`
	HasProvisionedETCD             bool      `json:"provisioned_etcd"`
	HasProvisionedMinion           bool      `json:"provisioned_minion"`
	HasProvisionedHA               bool      `json:"has_provisioned_ha"`
}

func (a *LightningMonkeyAgent) HasInitializedRoles() bool {
	return a.HasETCDRole || a.HasMasterRole || a.HasMinionRole || a.HasHARole
}

func (a *LightningMonkeyAgent) IsRunning() bool {
	if a.State == nil {
		return false
	}
	return time.Since(a.State.LastReportTime).Seconds() <= MaxAgentReportTimeoutSecs
}

func (a *Agent) IsRunning() bool {
	//"provisioning" phase is considered as running status which indicated that it's performing some initiative scripts.
	if (a.LastReportStatus == AgentStatus_Running || a.LastReportStatus == AgentStatus_Provisioning) && time.Since(a.LastReportTime).Seconds() <= MaxAgentReportTimeoutSecs {
		return true
	}
	//unhealthy or report timed out.
	return false
}

type AgentJob struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments"`
	Reason    string            `json:"reason"`
	//agent internal status listed blow.
	HadDone            bool
	HealthCheckHandler interface{}
}

type AgentStatus struct {
	ReportType   string `json:"report_type"`
	IP           string `json:"ip"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
	ReportTarget string `json:"report_target"`
}

type LightningMonkeyAgentReportStatus struct {
	IP      string                                          `json:"ip"`
	Items   map[string]LightningMonkeyAgentReportStatusItem `json:"items"`
	LeaseId int64                                           `json:"lease_id"`
}

type LightningMonkeyAgentReportStatusItem struct {
	HasProvisioned bool      `json:"has_provisioned"`
	Reason         string    `json:"reason"`
	LastSeenTime   time.Time `json:"last_seen_time"`
}

type KubernetesNodeInfo struct {
	NodeIP  string `json:"node_ip"`
	PodCIDR string `json:"node_cidr"`
}

type GenerateSystemRoutingRulesRequest struct {
	Nodes []KubernetesNodeInfo `json:"nodes"`
}

type ChangeClusterAndRolesRequest struct {
	OldClusterId string `json:"old_cluster_id"`
	NewClusterId string `json:"new_cluster_id"`
	IsETCDRole   bool   `json:"is_etcd_role"`
	IsMasterRole bool   `json:"is_master_role"`
	IsMinionRole bool   `json:"is_minion_role"`
	IsHARole     bool   `json:"is_ha_role"`
}
