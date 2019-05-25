package cache

import "github.com/g0194776/lightningmonkey/pkg/entities"

//go:generate mockgen -package=mock_lm -destination=../../mocks/mock_cluster_controller.go -source=cluster_controller.go ClusterController
type ClusterController interface {
	Dispose() //clean all of in use resource including backend watching jobs.
	GetSynchronizedRevision() int64
	GetStatus() entities.ClusterStatus
	GetClusterId() string
	GetCertificates() entities.LightningMonkeyCertificateCollection
	GetNextJob(currentAgentId string) (entities.AgentJob, error)
	SetSynchronizedRevision(id int64)
	SetCancellationFunc(f func()) //used for disposing in use resource.
	Lock()
	UnLock()
	OnAgentChanged(agent entities.LightningMonkeyAgent, isDeleted bool) error
	OnCertificateChanged(name string, cert string) error
}
