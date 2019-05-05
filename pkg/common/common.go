package common

import (
	"github.com/g0194776/lightningmonkey/pkg/controllers"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
)

var (
	StorageDriver              storage.StorageDriver
	ClusterStatementController *controllers.ClusterStatementController
	BasicImages                = map[string]entities.DockerImageCollection{
		"1.12.5": {
			DownloadType:      entities.DockerImageDownloadType_HTTP,
			HTTPDownloadToken: entities.HTTPDockerImageDownloadToken,
			Images: map[string]entities.DockerImage{
				"etcd":  {ImageName: "docker.io/mirrorgooglecontainers/etcd:3.2.24", DownloadAddr: "%s/apis/v1/registry/1.12.5/etcd.tar?token=%s"},
				"k8s":   {ImageName: "docker.io/mirrorgooglecontainers/hyperkube:v1.12.5", DownloadAddr: "%s/apis/v1/registry/1.12.5/hyperkube.tar?token=%s"},
				"infra": {ImageName: "docker.io/mirrorgooglecontainers/pause-amd64:3.1", DownloadAddr: "%s/apis/v1/registry/1.12.5/pause.tar?token=%s"},
			},
		},
	}
)
