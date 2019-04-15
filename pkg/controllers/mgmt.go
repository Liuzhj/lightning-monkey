package controllers

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/g0194776/lightningmonkey/pkg/strategies"
	"github.com/globalsign/mgo"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type ClusterStatementController struct {
	stopChan      chan int
	lockObj       *sync.Mutex
	storageDriver storage.StorageDriver
	clusters      map[string]*ClusterController
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
		csc.clusters = make(map[string]*ClusterController)
	}
	go csc.loadDatabase()
	go csc.updateProc()
}

func (csc *ClusterStatementController) loadDatabase() {
	var err error
	for {
		csc.lockObj.Lock()
		err = csc.innerLoadDatabase()
		if err != nil {
			logrus.Error(err)
		}
		csc.lockObj.Unlock()
		time.Sleep(time.Second * 10)
	}
}

func (csc *ClusterStatementController) innerLoadDatabase() error {
	clusters, err := csc.storageDriver.GetAllClusters()
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil
		}
		return fmt.Errorf("Failed to load all clusters information from remote database, error: %s", err.Error())
	}
	if clusters == nil || len(clusters) == 0 {
		return nil
	}
	for i := 0; i < len(clusters); i++ {
		agents, err := csc.storageDriver.GetAllAgentsByClusterId(clusters[i].Id.Hex())
		if err != nil {
			if err == mgo.ErrNotFound {
				continue
			}
			return err
		}
		if s, existed := csc.clusters[clusters[i].Id.Hex()]; !existed {
			c := ClusterController{}
			c.Initialize(csc.storageDriver, &strategies.DefaultClusterStatementStrategy{})
			c.Load(clusters[i], agents)
			csc.clusters[clusters[i].Id.Hex()] = &c
		} else {
			s.UpdateCache(agents)
		}
	}
	return nil
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
			err := csc.dumpToDatabase()
			if err != nil {
				logrus.Errorf("Failed to dump memory data to database, error: %s", err.Error())
			}
		}
		csc.lockObj.Unlock()
		time.Sleep(time.Second * 5)
	}
}

func (csc *ClusterStatementController) dumpToDatabase() error {
	var err error
	for _, strategy := range csc.clusters {
		agents := strategy.GetAgents()
		if agents == nil || len(agents) == 0 {
			continue
		}
		err = csc.storageDriver.BatchUpdateAgentStatus(agents)
		if err != nil {
			logrus.Errorf("Failed to dump memory data to database, cluster: %s, error: %s", strategy, err.Error())
		}
	}
	return nil
}

func (csc *ClusterStatementController) GetClusterStrategy(clusterId string) strategies.ClusterStatementStrategy {
	csc.lockObj.Lock()
	defer csc.lockObj.Unlock()
	return csc.clusters[clusterId]
}
