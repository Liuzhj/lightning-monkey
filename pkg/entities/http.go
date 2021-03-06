package entities

import (
	"time"
)

const (
	Succeed                          int = 0
	BizError                         int = 40000
	ParameterError                   int = 40001
	DeserializeError                 int = 40002
	UnsupportedError                 int = 40003
	AuthError                        int = 40004
	OperationFailed                  int = 40005
	ResourceHasBeingDeleted          int = 40006
	NotFound                         int = 40007
	InternalError                    int = 50000
	RESPONSEINFO                         = "HTTP_RESPONSE_INFO"
	DockerImageDownloadType_Registry     = "REGISTRY"
	DockerImageDownloadType_HTTP         = "HTTP"
)

var (
	HTTPDockerImageDownloadToken = ""
)

type Response struct {
	ErrorId     int    `json:"error-id"`
	Description string `json:"desc,omitempty"`
	Reason      string `json:"reason"`
	NeedCrash   bool   `json:"need_crash"`
}

type RegisterAgentResponse struct {
	Response
	AgentId        string                `json:"agent_id"`
	BasicImages    DockerImageCollection `json:"image_collection"`
	ClusterId      string                `json:"cluster_id"`
	MasterSettings map[string]string     `json:"master_settings"`
	LeaseId        int64                 `json:"lease_id"`
}

type CreateClusterResponse struct {
	Response
	ClusterId string `json:"cluster_id"`
}

type GetCertificateResponse struct {
	Response
	Content     string `json:"content"`
	ForceUpdate bool   `json:"force_update"`
}

type GetNextAgentJobResponse struct {
	Response
	Job *AgentJob `json:"job"`
}

type DockerImageCollection struct {
	DownloadType      string                 `json:"download_type"` //"REGISTRY", "HTTP"
	HTTPDownloadToken string                 `json:"http_download_token"`
	Images            map[string]DockerImage `json:"images"`
}

type DockerImage struct {
	DownloadAddr string `json:"download_addr"` //remote download address.
	ImageName    string `json:"image_name"`    //used for tagging a docker image as expected name on the local machine.
}

type AgentReportStatusResponse struct {
	Response
	LeaseId int64 `json:"lease_id"`
}

type GetClusterComponentStatusResponse struct {
	Response
	WatchPoints []WatchPoint `json:"status"`
}

type GetAgentListResponse struct {
	Response
	Agents []LightningMonkeyAgentBriefInformation `json:"agents"`
}

type LightningMonkeyAgentBriefInformation struct {
	HostInformation

	Id              string      `json:"id"`
	HasETCDRole     bool        `json:"has_etcd_role"`
	HasMasterRole   bool        `json:"has_master_role"`
	HasMinionRole   bool        `json:"has_minion_role"`
	HasHARole       bool        `json:"has_ha_role"`
	Hostname        string      `json:"hostname"`
	State           *AgentState `json:"state,omitempty"`
	DeploymentPhase int         `json:"deployment_phase"` //0-pending, 1-deploying, 2-deployed
}

type WatchPoint struct {
	IsSystemComponent bool      `json:"is_system_component"`
	Name              string    `json:"name"`
	Namespace         string    `json:"namespace"`
	Status            string    `json:"status"`
	LastCheckTime     time.Time `json:"last_check_time"`
}
