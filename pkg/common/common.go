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
			"etcd": "mirrorgooglecontainers/etcd",
			"k8s":  "mirrorgooglecontainers/hyperkube:v1.12.5"},
	}
)
