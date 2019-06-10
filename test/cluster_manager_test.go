package test

import (
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/g0194776/lightningmonkey/mocks"
	"github.com/g0194776/lightningmonkey/pkg/cache"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	assert "github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
	"unsafe"
)

type specTxn struct {
	rsp *clientv3.TxnResponse
	err error
}

func (txn *specTxn) If(cs ...clientv3.Cmp) clientv3.Txn {
	return txn
}

func (txn *specTxn) Then(ops ...clientv3.Op) clientv3.Txn {
	return txn
}

func (txn *specTxn) Else(ops ...clientv3.Op) clientv3.Txn {
	return txn
}

func (txn specTxn) SetResult(rsp *clientv3.TxnResponse, err error) clientv3.Txn {
	txn.rsp = rsp
	txn.err = err
	return &txn
}

func (txn *specTxn) Commit() (*clientv3.TxnResponse, error) {
	return txn.rsp, txn.err
}

func Test_InitWatch(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()
	sd := mock_lm.NewMockLightningMonkeyStorageDriver(gc)
	retChan := make(<-chan clientv3.WatchResponse)
	sd.EXPECT().Watch(gomock.Any(), "/lightning-monkey/clusters/", gomock.Any()).Return(retChan)
	cm := cache.ClusterManager{}
	err := cm.Initialize(sd)
	assert.Nil(t, err)
}

func Test_NewClusterBeingAdded(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()
	clusterId := uuid.NewV4().String()
	sd := mock_lm.NewMockLightningMonkeyStorageDriver(gc)
	retChan := make(chan clientv3.WatchResponse)
	//used for watching clusters changes.
	sd.EXPECT().Watch(gomock.Any(), "/lightning-monkey/clusters/", gomock.Any()).Return(retChan)
	//used for watching agents & certifications changes.
	sd.EXPECT().Watch(gomock.Any(), "/lightning-monkey/clusters/"+clusterId+"/", gomock.Any()).Return(retChan)
	duration, _ := time.ParseDuration("5s")
	sd.EXPECT().GetRequestTimeoutDuration().Return(duration).AnyTimes()
	sd.EXPECT().Txn(gomock.Any()).Return(specTxn{}.SetResult(nil, nil)).Times(3) //three sub-keys needed to check.
	sd.EXPECT().Get(gomock.Any(), "/lightning-monkey/clusters/"+clusterId+"/agents/", gomock.Any()).Return(&clientv3.GetResponse{
		Header: &etcdserverpb.ResponseHeader{Revision: 888},
	}, nil)
	cm := cache.ClusterManager{}
	err := cm.Initialize(sd)
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		fmt.Println("Trigger an event for cluster...")
		clusterSettings := entities.LightningMonkeyClusterSettings{
			Id:                clusterId,
			CreateTime:        time.Now(),
			Name:              "cluster-1",
			ExpectedETCDCount: 3,
			ServiceCIDR:       "10.254.0.0/16",
			KubernetesVersion: "1.12.5",
			PodNetworkCIDR:    "172.1.0.0/16",
			SecurityToken:     "",
			ServiceDNSDomain:  ".cluster.local",
			NetworkStack: &entities.NetworkStackSettings{
				Type: entities.NetworkStack_KubeRouter,
			},
		}
		value, err := json.Marshal(clusterSettings)
		if err != nil {
			panic(err)
		}
		retChan <- clientv3.WatchResponse{
			Events: []*clientv3.Event{
				&clientv3.Event{
					Kv: &mvccpb.KeyValue{
						Key:   []byte("/lightning-monkey/clusters/" + clusterId + "/metadata"),
						Value: []byte(value),
					}},
			},
		}
		time.Sleep(time.Second * 3)
		wg.Done()
	}()
	fmt.Println("Waiting...")
	wg.Wait()
	//unsafely retrieve internal clusters collection for checking state.
	m := *(*map[string]cache.ClusterController)(unsafe.Pointer(uintptr(unsafe.Pointer(&cm)) + 8))
	assert.True(t, len(m) == 1)
	assert.True(t, m[clusterId] != nil)
}

func Test_NewClusterAndOneOfflineAgentBeingFound(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()
	clusterId := uuid.NewV4().String()
	agentId := uuid.NewV4().String()
	sd := mock_lm.NewMockLightningMonkeyStorageDriver(gc)
	retChan := make(chan clientv3.WatchResponse)
	//used for watching clusters changes.
	sd.EXPECT().Watch(gomock.Any(), "/lightning-monkey/clusters/", gomock.Any()).Return(retChan)
	//used for watching agents & certifications changes.
	sd.EXPECT().Watch(gomock.Any(), "/lightning-monkey/clusters/"+clusterId+"/", gomock.Any()).Return(retChan)
	duration, _ := time.ParseDuration("5s")
	sd.EXPECT().GetRequestTimeoutDuration().Return(duration).AnyTimes()
	sd.EXPECT().Txn(gomock.Any()).Return(specTxn{}.SetResult(nil, nil)).Times(3) //three sub-keys needed to check.
	agent := entities.LightningMonkeyAgent{
		Id:            clusterId,
		ClusterId:     clusterId,
		MetadataId:    uuid.NewV4().String(),
		Hostname:      "keppers",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	value, err := json.Marshal(agent)
	if err != nil {
		panic(err)
	}
	sd.EXPECT().Get(gomock.Any(), "/lightning-monkey/clusters/"+clusterId+"/agents/", gomock.Any()).Return(&clientv3.GetResponse{
		Header: &etcdserverpb.ResponseHeader{Revision: 888},
		Kvs: []*mvccpb.KeyValue{
			{
				Key:   []byte("/lightning-monkey/clusters/" + clusterId + "/agents/" + agentId + "/settings"),
				Value: []byte(value),
			},
		},
		Count: 1,
	}, nil)
	cm := cache.ClusterManager{}
	err = cm.Initialize(sd)
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		fmt.Println("Trigger an event for cluster...")
		clusterSettings := entities.LightningMonkeyClusterSettings{
			Id:                clusterId,
			CreateTime:        time.Now(),
			Name:              "cluster-1",
			ExpectedETCDCount: 3,
			ServiceCIDR:       "10.254.0.0/16",
			KubernetesVersion: "1.12.5",
			PodNetworkCIDR:    "172.1.0.0/16",
			SecurityToken:     "",
			ServiceDNSDomain:  ".cluster.local",
			NetworkStack: &entities.NetworkStackSettings{
				Type: entities.NetworkStack_KubeRouter,
			},
		}
		value, err := json.Marshal(clusterSettings)
		if err != nil {
			panic(err)
		}
		retChan <- clientv3.WatchResponse{
			Events: []*clientv3.Event{
				&clientv3.Event{
					Kv: &mvccpb.KeyValue{
						Key:   []byte("/lightning-monkey/clusters/" + clusterId + "/metadata"),
						Value: []byte(value),
					}},
			},
		}
		time.Sleep(time.Second * 3)
		wg.Done()
	}()
	fmt.Println("Waiting...")
	wg.Wait()
	//unsafely retrieve internal clusters collection for checking state.
	m := *(*map[string]cache.ClusterController)(unsafe.Pointer(uintptr(unsafe.Pointer(&cm)) + 8))
	assert.True(t, len(m) == 1)
	assert.True(t, m[clusterId] != nil)
	//we'd triggered an agent event without any state, that's why we get nothing here.
	assert.True(t, m[clusterId].GetTotalCountByRole(entities.AgentRole_ETCD) == 0)
}

func Test_NewClusterAndOneOnlineAgentBeingFound(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()
	clusterId := uuid.NewV4().String()
	agentId := uuid.NewV4().String()
	sd := mock_lm.NewMockLightningMonkeyStorageDriver(gc)
	retChan := make(chan clientv3.WatchResponse)
	//used for watching clusters changes.
	sd.EXPECT().Watch(gomock.Any(), "/lightning-monkey/clusters/", gomock.Any()).Return(retChan)
	//used for watching agents & certifications changes.
	sd.EXPECT().Watch(gomock.Any(), "/lightning-monkey/clusters/"+clusterId+"/", gomock.Any()).Return(retChan)
	duration, _ := time.ParseDuration("5s")
	sd.EXPECT().GetRequestTimeoutDuration().Return(duration).AnyTimes()
	sd.EXPECT().Txn(gomock.Any()).Return(specTxn{}.SetResult(nil, nil)).Times(3) //three sub-keys needed to check.
	agent := entities.LightningMonkeyAgent{
		Id:            clusterId,
		ClusterId:     clusterId,
		MetadataId:    uuid.NewV4().String(),
		Hostname:      "keppers",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	value, err := json.Marshal(agent)
	if err != nil {
		panic(err)
	}
	state := entities.AgentState{
		LastReportIP:       "127.0.0.1",
		LastReportTime:     time.Now(),
		HasProvisionedETCD: true,
	}
	value2, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	sd.EXPECT().Get(gomock.Any(), "/lightning-monkey/clusters/"+clusterId+"/agents/", gomock.Any()).Return(&clientv3.GetResponse{
		Header: &etcdserverpb.ResponseHeader{Revision: 888},
		Kvs: []*mvccpb.KeyValue{
			{
				Key:   []byte("/lightning-monkey/clusters/" + clusterId + "/agents/" + agentId + "/settings"),
				Value: []byte(value),
			},
			{
				Key:   []byte("/lightning-monkey/clusters/" + clusterId + "/agents/" + agentId + "/state"),
				Value: []byte(value2),
			},
		},
		Count: 1,
	}, nil)
	cm := cache.ClusterManager{}
	err = cm.Initialize(sd)
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		fmt.Println("Trigger an event for cluster...")
		clusterSettings := entities.LightningMonkeyClusterSettings{
			Id:                clusterId,
			CreateTime:        time.Now(),
			Name:              "cluster-1",
			ExpectedETCDCount: 3,
			ServiceCIDR:       "10.254.0.0/16",
			KubernetesVersion: "1.12.5",
			PodNetworkCIDR:    "172.1.0.0/16",
			SecurityToken:     "",
			ServiceDNSDomain:  ".cluster.local",
			NetworkStack: &entities.NetworkStackSettings{
				Type: entities.NetworkStack_KubeRouter,
			},
		}
		value, err := json.Marshal(clusterSettings)
		if err != nil {
			panic(err)
		}
		retChan <- clientv3.WatchResponse{
			Events: []*clientv3.Event{
				&clientv3.Event{
					Kv: &mvccpb.KeyValue{
						Key:   []byte("/lightning-monkey/clusters/" + clusterId + "/metadata"),
						Value: []byte(value),
					}},
			},
		}
		time.Sleep(time.Second * 3)
		wg.Done()
	}()
	fmt.Println("Waiting...")
	wg.Wait()
	//unsafely retrieve internal clusters collection for checking state.
	m := *(*map[string]cache.ClusterController)(unsafe.Pointer(uintptr(unsafe.Pointer(&cm)) + 8))
	assert.True(t, len(m) == 1)
	assert.True(t, m[clusterId] != nil)
	assert.True(t, m[clusterId].GetTotalCountByRole(entities.AgentRole_ETCD) == 1)
	assert.True(t, m[clusterId].GetTotalCountByRole(entities.AgentRole_Master) == 0)
	assert.True(t, m[clusterId].GetTotalCountByRole(entities.AgentRole_Minion) == 0)
}
