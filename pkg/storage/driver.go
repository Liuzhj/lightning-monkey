package storage

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage/mongodb"
	"strings"
	"time"
)

type StorageDriver interface {
	Initialize(args map[string]string) error
	GetCluster(clusterId string) (*entities.Cluster, error)
	SaveCluster(cluster *entities.Cluster, certsMap *certs.GeneratedCertsMap) error
	GetCertificatesByClusterId(clusterId string) (entities.CertificateCollection, error)
	GetCertificatesByClusterIdAndName(clusterId string, name string) (*entities.Certificate, error)
	GetAllClusters() ([]*entities.Cluster, error)
	GetAllAgentsByClusterId(clusterId string) ([]*entities.Agent, error)
	GetAgentByMetadataId(metadataId string) (*entities.Agent, error)
	SaveAgent(agent *entities.Agent) error
	SaveCertificateToCluster(cluster *entities.Cluster, certsMap *certs.GeneratedCertsMap) error
	UpdateCluster(cluster *entities.Cluster) error
	UpdateAgentStatus(agent *entities.Agent) error
	BatchUpdateAgentStatus(agents []*entities.Agent) error
}

//go:generate mockgen -package=mock_lm -destination=../../mocks/mock_driver.go -source=driver.go LightningMonkeyStorageDriver
type LightningMonkeyStorageDriver interface {
	Initialize(settings map[string]string) error
	GetRequestTimeoutDuration() time.Duration
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan
	Txn(ctx context.Context) clientv3.Txn
}

type StorageDriverFactory struct {
}

func (sdf *StorageDriverFactory) NewStorageDriver(t string) (StorageDriver, error) {
	switch strings.ToLower(t) {
	case "mongo":
		return &mongodb.MongoDBStorageDriver{}, nil
	case "etcd":
	}
	return nil, fmt.Errorf("Unsupported storage driver: %s", t)
}
