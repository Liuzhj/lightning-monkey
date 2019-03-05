package entities

const (
	Succeed                 int = 0
	BizError                int = 40000
	ParameterError          int = 40001
	DeserializeError        int = 40002
	UnsupportedError        int = 40003
	AuthError               int = 40004
	OperationFailed         int = 40005
	ResourceHasBeingDeleted int = 40006
	NotFound                int = 40007
	InternalError           int = 50000
	RESPONSEINFO                = "HTTP_RESPONSE_INFO"
)

type Response struct {
	ErrorId     int    `json:"error-id"`
	Description string `json:"desc,omitempty"`
	Reason      string `json:"reason"`
}

type CreateClusterResponse struct {
	Response
	Cluster *Cluster `json:"cluster"`
}

type GetCertificateResponse struct {
	Response
	Content string `json:"content"`
}
