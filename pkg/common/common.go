package common

import (
	"github.com/g0194776/lightningmonkey/pkg/controllers"
	"github.com/g0194776/lightningmonkey/pkg/storage"
)

var (
	StorageDriver              storage.StorageDriver
	ClusterStatementController *controllers.ClusterStatementController
	BasicImages                = map[string]map[string]string{
		"1.12.5": {
			"etcd":  "docker.io/mirrorgooglecontainers/etcd:3.2.24",
			"k8s":   "docker.io/mirrorgooglecontainers/hyperkube:v1.12.5",
			"infra": "docker.io/mirrorgooglecontainers/pause-amd64:3.1",
		},
	}
)