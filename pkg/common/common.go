package common

import (
	"github.com/g0194776/lightningmonkey/pkg/cache"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
)

var (
	StorageDriver  storage.LightningMonkeyStorageDriver
	ClusterManager *cache.ClusterManager
	BasicImages    = map[string]entities.DockerImageCollection{
		"1.12.5": {
			DownloadType:      entities.DockerImageDownloadType_HTTP,
			HTTPDownloadToken: "",
			Images: map[string]entities.DockerImage{
				"etcd":  {ImageName: "docker.io/mirrorgooglecontainers/etcd:3.2.24", DownloadAddr: "%s/apis/v1/registry/1.12.5/etcd.tar?token=%s"},
				"k8s":   {ImageName: "g0194776/lightning-monkey-hyperkube:v1.12.5-2", DownloadAddr: "%s/apis/v1/registry/1.12.5/k8s.tar?token=%s"},
				"infra": {ImageName: "docker.io/mirrorgooglecontainers/pause-amd64:3.1", DownloadAddr: "%s/apis/v1/registry/1.12.5/infra.tar?token=%s"},
			},
		},
	}
)
