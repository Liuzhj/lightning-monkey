package strategies

import (
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ClusterStatementController struct {
	stopChan      chan int
	lockObj       *sync.Mutex
	storageDriver storage.StorageDriver
	clusters      map[string]ClusterStatementStrategy
}

func (csc *ClusterStatementController) Initialize(storageDriver storage.StorageDriver) {
	csc.storageDriver = storageDriver
}

func (csc *ClusterStatementController) Start() {
	if csc.lockObj == nil {
		csc.lockObj = &sync.Mutex{}
	}
	if csc.stopChan == nil {
		csc.stopChan = make(chan int, 1)
	}
	if csc.clusters == nil {
		csc.clusters = make(map[string]ClusterStatementStrategy)
	}
	go csc.updateProc()
}

func (csc *ClusterStatementController) Stop() {
	csc.lockObj.Lock()
	defer csc.lockObj.Unlock()
	close(csc.stopChan)
	csc.stopChan = nil
}

func (csc *ClusterStatementController) updateProc() {
	for {
		csc.lockObj.Lock()
		select {
		case _, isOpen := <-csc.stopChan:
			if !isOpen {
				csc.lockObj.Unlock()
				return
			}
		default:
			mapping, err := csc.getClustersInformation()
			if err != nil {
				logrus.Error(err.Error())
				break
			}
			csc.clusters = mapping
		}
		csc.lockObj.Unlock()
		time.Sleep(time.Second * 5)
	}
}

func (csc *ClusterStatementController) getClustersInformation() (map[string]ClusterStatementStrategy, error) {
	mapping := make(map[string]ClusterStatementStrategy)
	clusters, err := csc.storageDriver.GetAllClusters()
	if err != nil {
		return mapping, err
	}
	if clusters == nil || len(clusters) == 0 {
		return mapping, nil
	}
	for i := 0; i < len(clusters); i++ {
		clusterId := clusters[i].Id.Hex()
		agents, err := csc.storageDriver.GetAllAgentsByClusterId(clusterId)
		if err != nil {
			return mapping, err
		}
		strategy := &DefaultClusterStatementStrategy{}
		strategy.Load(clusters[i], agents)
		mapping[clusterId] = strategy
	}
	return mapping, nil
}

func (csc *ClusterStatementController) GetClusterStrategy(clusterId string) ClusterStatementStrategy {
	csc.lockObj.Lock()
	defer csc.lockObj.Unlock()
	return csc.clusters[clusterId]
}
