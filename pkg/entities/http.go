package entities

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
	BasicImages    DockerImageCollection `json:"image_collection"`
	MasterSettings map[string]string     `json:"master_settings"`
}

type CreateClusterResponse struct {
	Response
	Cluster *Cluster `json:"cluster"`
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
