package cache

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

//go:generate mockgen -package=mock_lm -destination=../../mocks/mock_cluster_controller.go -source=cluster_controller.go ClusterController
type ClusterController interface {
	Dispose() //clean all of in use resource including backend watching jobs.
	GetSynchronizedRevision() int64
	GetStatus() string
	GetClusterId() string
	GetCertificates() entities.LightningMonkeyCertificateCollection
	GetNextJob(agent entities.LightningMonkeyAgent) (entities.AgentJob, error)
	GetTotalCountByRole(role string) int
	GetTotalProvisionedCountByRole(role string) int
	GetSettings() entities.LightningMonkeyClusterSettings
	GetCachedAgent(agentId string) (*entities.LightningMonkeyAgent, error)
	Initialize()
	SetSynchronizedRevision(id int64)
	SetCancellationFunc(f func()) //used for disposing in use resource.
	Lock()
	UnLock()
	UpdateClusterSettings(settings entities.LightningMonkeyClusterSettings) ClusterController
	OnAgentChanged(agent entities.LightningMonkeyAgent, isDeleted bool) error
	OnCertificateChanged(name string, cert string, isDeleted bool) error
}

type ClusterControllerImple struct {
	cache                *AgentCache
	certs                map[string]string
	cancellationFunc     func()
	jobScheduler         ClusterJobScheduler
	isDisposed           uint32
	lockObj              *sync.Mutex
	settings             entities.LightningMonkeyClusterSettings
	synchronizedRevision int64
}

func (cc *ClusterControllerImple) GetSettings() entities.LightningMonkeyClusterSettings {
	return cc.settings
}

//used for debugging internal state.
func (cc *ClusterControllerImple) GetTotalCountByRole(role string) int {
	return cc.cache.GetTotalCountByRole(role)
}

//used for debugging internal state.
func (cc *ClusterControllerImple) GetTotalProvisionedCountByRole(role string) int {
	return cc.cache.GetTotalProvisionedCountByRole(role)
}

func (cc *ClusterControllerImple) Initialize() {
	if cc.lockObj == nil {
		cc.lockObj = &sync.Mutex{}
	}
	cc.certs = make(map[string]string)
	cc.cache = &AgentCache{}
	cc.cache.Initialize()
	cc.jobScheduler = &ClusterJobSchedulerImple{}
	cc.jobScheduler.InitializeStrategies()
}

func (cc *ClusterControllerImple) Dispose() {
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("Occurred an unhandled exception during disposing cluster controller, cluster-id: %s, error: %v", cc.settings.Id, err)
		}
	}()
	atomic.StoreUint32(&cc.isDisposed, 1)
	if cc.cancellationFunc != nil {
		cc.cancellationFunc()
	}
	cc.cache = nil
}

func (cc *ClusterControllerImple) GetSynchronizedRevision() int64 {
	return cc.synchronizedRevision
}

func (cc *ClusterControllerImple) GetStatus() string {
	panic("implement me")
}

func (cc *ClusterControllerImple) GetClusterId() string {
	return cc.settings.Id
}

func (cc *ClusterControllerImple) GetCertificates() entities.LightningMonkeyCertificateCollection {
	collection := make([]*entities.CertificateKeyPair, 0, len(cc.certs))
	i := 0
	for k, v := range cc.certs {
		collection[i] = &entities.CertificateKeyPair{Name: k, Value: v}
		i++
	}
	return collection
}

func (cc *ClusterControllerImple) GetNextJob(agent entities.LightningMonkeyAgent) (entities.AgentJob, error) {
	return cc.jobScheduler.GetNextJob(cc.settings, agent, cc.cache)
}

func (cc *ClusterControllerImple) SetSynchronizedRevision(id int64) {
	cc.synchronizedRevision = id
}

func (cc *ClusterControllerImple) SetCancellationFunc(f func()) {
	cc.cancellationFunc = f
}

func (cc *ClusterControllerImple) Lock() {
	cc.lockObj.Lock()
}

func (cc *ClusterControllerImple) UnLock() {
	cc.lockObj.Unlock()
}

func (cc *ClusterControllerImple) OnAgentChanged(agent entities.LightningMonkeyAgent, isDeleted bool) error {
	if atomic.LoadUint32(&cc.isDisposed) == 1 {
		return fmt.Errorf("Cannot update cache to a disposed cluster controller, cluster-id: %s", cc.settings.Id)
	}
	if isDeleted || agent.State == nil {
		cc.cache.Offline(agent)
	} else {
		cc.cache.Online(agent)
	}
	return nil
}

func (cc *ClusterControllerImple) OnCertificateChanged(name string, cert string, isDeleted bool) error {
	if atomic.LoadUint32(&cc.isDisposed) == 1 {
		return fmt.Errorf("Cannot update cache to a disposed cluster controller, cluster-id: %s", cc.settings.Id)
	}
	if isDeleted {
		delete(cc.certs, name)
		return nil
	}
	cc.certs[name] = cert
	return nil
}

func (cc *ClusterControllerImple) UpdateClusterSettings(settings entities.LightningMonkeyClusterSettings) ClusterController {
	cc.settings = settings
	return cc
}

func (cc *ClusterControllerImple) GetCachedAgent(agentId string) (*entities.LightningMonkeyAgent, error) {
	if atomic.LoadUint32(&cc.isDisposed) == 1 {
		return nil, fmt.Errorf("Cannot update cache to a disposed cluster controller, cluster-id: %s", cc.settings.Id)
	}
	cc.cache.Lock()
	defer cc.cache.Unlock()
	var isOK bool
	var agent *entities.LightningMonkeyAgent
	if agent, isOK = cc.cache.etcd[agentId]; isOK {
		return agent, nil
	}
	if agent, isOK = cc.cache.k8sMaster[agentId]; isOK {
		return agent, nil
	}
	if agent, isOK = cc.cache.k8sMinion[agentId]; isOK {
		return agent, nil
	}
	return nil, fmt.Errorf("Agent: %s not found!", agentId)
}
