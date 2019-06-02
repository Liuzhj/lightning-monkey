package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
)

type ClusterManager struct {
	lockObj       *sync.Mutex
	clusters      map[string]ClusterController
	storageDriver storage.LightningMonkeyStorageDriver
}

func (cm *ClusterManager) Initialize(storageDriver storage.LightningMonkeyStorageDriver) error {
	if cm.lockObj == nil {
		cm.lockObj = &sync.Mutex{}
	}
	cm.clusters = make(map[string]ClusterController)
	cm.storageDriver = storageDriver
	return cm.watchClusterChanges()
}

func (cm *ClusterManager) GetClusterCertificateByName(clusterId string, certName string) (string, error) {
	var isOK bool
	var cluster ClusterController
	cm.lockObj.Lock()
	cluster, isOK = cm.clusters[clusterId]
	cm.lockObj.Unlock()
	if !isOK {
		return "", fmt.Errorf("Cluster %s not found!", clusterId)
	}
	return cluster.GetCertificates().GetCertificateContent(certName), nil
}

func (cm *ClusterManager) Register(cc ClusterController) error {
	clusterId := cc.GetClusterId()
	if clusterId == "" {
		return errors.New("Failed to register cluster instance to cluster manager, cluster identity cannot be empty!")
	}
	err := cm.doRegister(cc)
	if err != nil {
		return err
	}
	cm.lockObj.Lock()
	cm.clusters[clusterId] = cc
	cm.lockObj.Unlock()
	return nil
}

func (cm *ClusterManager) doRegister(cc ClusterController) error {
	//STEP 1, add new KEY on specific path of remote ETCD if not exists.
	clusterPath := fmt.Sprintf("/lightning-monkey/clusters/%s", cc.GetClusterId())
	agentsPath := clusterPath + "/agents"
	keys := []string{
		clusterPath,
		agentsPath,
		clusterPath + "/certificates"}
	var err error
	for i := 0; i < len(keys); i++ {
		err = cm.createKeyIfNotExists(keys[i], "")
		if err != nil {
			return fmt.Errorf("Failed to perform works with remote ETCD, Key: %s, errors: %s", keys[i], err.Error())
		}
	}
	//STEP 2, do watch all resource changes for given cluster identity.
	ctx, cancel := context.WithTimeout(context.Background(), cm.storageDriver.GetRequestTimeoutDuration())
	defer cancel()
	rsp, err := cm.storageDriver.Get(ctx, agentsPath+"/", clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("Failed to get specified Key's(%s) value from remote ETCD server, error: %s", agentsPath, err.Error())
	}
	//lock current cluster until finished cache synchronization.
	cc.Lock()
	//set received revision as cache version.
	cc.SetSynchronizedRevision(rsp.Header.Revision)
	if rsp.Count > 0 {
		agents := make(map[string]*entities.LightningMonkeyAgent)
		for i := 0; i < len(rsp.Kvs); i++ {
			subKeys := strings.FieldsFunc(string(rsp.Kvs[i].Key), func(r rune) bool {
				return r == '/'
			})
			//i.e. "/lightning-monkey/clusters/sjh23897ehj387e/agents/1hs73jkd83ponf874/settings"
			//ETCD always returns ordered result set, that's why we needn't use another one collection to ensures that no any agent are missed.
			if subKeys[0] == "lightning-monkey" && subKeys[1] == "clusters" && subKeys[3] == "agents" {
				if subKeys[len(subKeys)-1] == "settings" {
					//new agent.
					a := entities.LightningMonkeyAgent{}
					err = json.Unmarshal(rsp.Kvs[i].Value, &a)
					if err != nil {
						logrus.Errorf("Failed to unmarshal JSON formatted data to Lightning Monkey agent object, error: %s", err.Error())
						continue
					}
					agents[subKeys[4]] = &a
				} else if subKeys[len(subKeys)-1] == "state" {
					//agent's state.
					s := entities.AgentState{}
					err = json.Unmarshal(rsp.Kvs[i].Value, &s)
					if err != nil {
						logrus.Errorf("Failed to unmarshal JSON formatted data to Lightning Monkey agent state object, error: %s", err.Error())
						continue
					}
					//considered that agent.State is a lease-guaranteed object, we don't care dirty data here.
					if a, isOK := agents[subKeys[4]]; isOK {
						a.State = &s
					}
				}
			}
		}
		//trigger notification that new agents has been being found here, that's the first time to update hot cache.
		if len(agents) > 0 {
			for _, agent := range agents {
				_ = cc.OnAgentChanged(*agent, false)
			}
		}
	}
	cc.UnLock()
	//OKey, all of previous actions has done, we need to watch all of subsequent events...
	wc := cm.storageDriver.Watch(context.Background(), clusterPath+"/" /*agent & certificate changes are included*/, clientv3.WithPrefix())
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	cc.SetCancellationFunc(cancelFunc)
	go cm.watchChanges(cancelCtx, wc, cc)
	return nil
}

func (cm *ClusterManager) watchChanges(ctx context.Context, wc clientv3.WatchChan, cc ClusterController) {
	var err error
	var changed bool
	var agentId string
	var agent entities.LightningMonkeyAgent
	for {
		select {
		case <-ctx.Done():
			return
		case rsp, isOK := <-wc:
			if !isOK {
				return
			}
			if rsp.Events == nil || len(rsp.Events) == 0 {
				continue
			}
			if rsp.Header.Revision <= cc.GetSynchronizedRevision() {
				logrus.Debugf("Ignored ETCD event, revision: %d, It's behind of cluster latest revision: %d!", rsp.Header.Revision, cc.GetSynchronizedRevision())
				continue
			}
			for i := 0; i < len(rsp.Events); i++ {
				logrus.Debugf("Received ETCD event: Event=%d, Key=%s", rsp.Events[i].Type, string(rsp.Events[i].Kv.Key))
				subKeys := strings.FieldsFunc(string(rsp.Events[i].Kv.Key), func(r rune) bool {
					return r == '/'
				})
				//detect agents changes.
				if agentId, changed = isAgentChanged(subKeys); changed {
					agent, err = cm.GetAgentFromETCD(cc.GetClusterId(), agentId)
					if err != nil {
						logrus.Errorf("Failed to retrieve newest version of Lightning Monkey's Agent data from remote ETCD, error: %s", err.Error())
						continue
					}
					cc.Lock()
					err = cc.OnAgentChanged(agent, rsp.Events[i].Type == clientv3.EventTypeDelete)
					cc.UnLock()
					if err != nil {
						logrus.Errorf("Failed to update hot cache for cluster: %s, error: %s", cc.GetClusterId(), err.Error())
						continue
					}
				}
				//detect certificates changes.
				if isCertificatesChanged(subKeys) {
					cert := string(rsp.Events[i].Kv.Value)
					if cert == "" {
						logrus.Errorf("Illegal certificate content being received from remote ETCD event, key: %s", string(rsp.Events[i].Kv.Key))
						continue
					}
					cc.Lock()
					err = cc.OnCertificateChanged(string(rsp.Events[i].Kv.Key), cert, rsp.Events[i].Type == clientv3.EventTypeDelete)
					cc.UnLock()
					if err != nil {
						logrus.Errorf("Failed to update hot cache with certificate changes, cluster: %s, key: %s error: %s", cc.GetClusterId(), string(rsp.Events[i].Kv.Key), err.Error())
						continue
					}
				}
			}
		}
	}
}

func (cm *ClusterManager) GetAgentFromETCD(clusterId, agentId string) (entities.LightningMonkeyAgent, error) {
	agent := entities.LightningMonkeyAgent{}
	settingsPath := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/settings", clusterId, agentId)
	ctx, cancel := context.WithTimeout(context.Background(), cm.storageDriver.GetRequestTimeoutDuration())
	defer cancel()
	rsp, err := cm.storageDriver.Get(ctx, settingsPath)
	if err != nil {
		return agent, err
	}
	if rsp.Count == 0 {
		return agent, fmt.Errorf("Key: %s not found!", settingsPath)
	}
	err = json.Unmarshal(rsp.Kvs[0].Value, &agent)
	if err != nil {
		return agent, err
	}
	statePath := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/state", clusterId, agentId)
	ctx2, cancel2 := context.WithTimeout(context.Background(), cm.storageDriver.GetRequestTimeoutDuration())
	defer cancel2()
	rsp, err = cm.storageDriver.Get(ctx2, statePath)
	if err != nil {
		return agent, err
	}
	if rsp.Count == 0 {
		//ignored missed state object, it's lease-guaranteed.
		return agent, nil
	}
	state := entities.AgentState{}
	err = json.Unmarshal(rsp.Kvs[0].Value, &state)
	if err != nil {
		return agent, err
	}
	agent.State = &state
	return agent, nil
}

func (cm *ClusterManager) createKeyIfNotExists(path string, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cm.storageDriver.GetRequestTimeoutDuration())
	defer cancel()
	_, err := cm.storageDriver.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(path), "=", 0)).
		Then(clientv3.OpPut(path, value)).
		Commit()
	return err
}

func (cm *ClusterManager) watchClusterChanges() error {
	wc := cm.storageDriver.Watch(context.Background(), "/lightning-monkey/clusters/", clientv3.WithPrefix())
	go func() {
		var isOK bool
		var wr clientv3.WatchResponse
		for {
			wr, isOK = <-wc
			if !isOK {
				return
			}
			if len(wr.Events) == 0 {
				continue
			}
			for i := 0; i < len(wr.Events); i++ {
				subKeys := strings.FieldsFunc(string(string(wr.Events[i].Kv.Key)), func(r rune) bool {
					return r == '/'
				})
				var isChange bool
				var clusterId string
				if clusterId, isChange = isClusterChanged(subKeys); !isChange {
					continue
				}
				err := cm.doClusterChange(clusterId, wr.Events[i].Kv.Value, wr.Events[i].Type == clientv3.EventTypeDelete)
				if err != nil {
					logrus.Errorf("Failed to handle cluster-level changes, key: %s, error: %s", string(wr.Events[i].Kv.Key), err.Error())
				}
			}
		}
	}()
	return nil
}

func (cm *ClusterManager) doClusterChange(clusterId string, value []byte, isDeleted bool) error {
	var isOK bool
	var cluster ClusterController
	cm.lockObj.Lock()
	defer cm.lockObj.Unlock()
	cluster, isOK = cm.clusters[clusterId]
	if isDeleted {
		if !isOK {
			return nil
		}
		cluster.Dispose()
		delete(cm.clusters, clusterId)
		logrus.Debugf("Cluster %s had been disposed by deletion event!", clusterId)
		return nil
	}
	settings := entities.LightningMonkeyClusterSettings{}
	err := json.Unmarshal(value, &settings)
	if err != nil {
		return err
	}
	//create new cluster to cache if not exists.
	if !isOK {
		cluster = ClusterControllerImple{}.UpdateClusterSettings(settings)
		cluster.Initialize()
		err = cm.doRegister(cluster)
		if err != nil {
			return err
		}
		cm.clusters[clusterId] = cluster
		logrus.Debugf("Registered new cluster: %s", cluster.GetClusterId())
		return nil
	}
	//update cache.
	cluster.Lock()
	cluster.UpdateClusterSettings(settings)
	cluster.UnLock()
	logrus.Debugf("Updated cluster %s settings!", cluster.GetClusterId())
	return nil
}

func isAgentChanged(subKeys []string) (string /*parsed agent id*/, bool) {
	if subKeys[0] == "lightning-monkey" && subKeys[1] == "clusters" && subKeys[3] == "agents" {
		if subKeys[len(subKeys)-1] == "settings" || subKeys[len(subKeys)-1] == "state" {
			return subKeys[len(subKeys)-2], true
		}
	}
	return "", false
}

func isClusterChanged(subKeys []string) (string /*parsed cluster id*/, bool) {
	if subKeys[0] == "lightning-monkey" && subKeys[1] == "clusters" && subKeys[len(subKeys)-1] == "metadata" {
		return subKeys[len(subKeys)-2], true
	}
	return "", false
}

func isCertificatesChanged(subKeys []string) bool {
	if subKeys[0] == "lightning-monkey" && subKeys[1] == "clusters" && subKeys[len(subKeys)-1] == "certificates" {
		return true
	}
	return false
}
