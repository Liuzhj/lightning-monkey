package storage

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/storage/mongodb"
	"strings"
)

type StorageDriver interface {
	Initialize(args map[string]string) error
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
