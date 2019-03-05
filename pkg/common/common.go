package common

import (
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/g0194776/lightningmonkey/pkg/strategies"
)

var (
	StorageDriver              storage.StorageDriver
	ClusterStatementController *strategies.ClusterStatementController
)
