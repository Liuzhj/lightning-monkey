package common

import (
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/g0194776/lightningmonkey/pkg/strategies"
)

var (
	StorageDriver              storage.StorageDriver
	ClusterStatementController *strategies.ClusterStatementController
	BasicImages                = map[string]map[string]string{
		"1.12.5": {
			"etcd":  "docker.io/mirrorgooglecontainers/etcd",
			"k8s":   "docker.io/mirrorgooglecontainers/hyperkube:v1.12.5",
			"infra": "docker.io/mirrorgooglecontainers/pause-amd64:3.0",
		},
	}
)
