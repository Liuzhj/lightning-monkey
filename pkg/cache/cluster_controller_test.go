package cache

import (
	"fmt"
	mock_lm "github.com/g0194776/lightningmonkey/mocks"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	assert "github.com/stretchr/testify/require"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/etcdserverpb"
	"testing"
)

func Test_AgentOnline(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()
	clusterId := uuid.NewV4().String()
	sd := mock_lm.NewMockLightningMonkeyStorageDriver(gc)
	//full-sync logic
	sd.EXPECT().Get(gomock.Any(), fmt.Sprintf("/lightning-monkey/clusters/%s/", clusterId), gomock.Any()).Return(&clientv3.GetResponse{
		Header: &etcdserverpb.ResponseHeader{Revision: 0},
	}, nil)

	cc := ClusterControllerImple{settings: entities.LightningMonkeyClusterSettings{
		Id: clusterId,
	}}
	cc.Initialize(sd)
	agent1 := entities.LightningMonkeyAgent{
		Id:            uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			HasProvisionedETCD: true,
		},
	}
	err := cc.OnAgentChanged(agent1, false)
	assert.Nil(t, err)
	assert.True(t, len(cc.cache.etcd) == 1)
}

//func Test_AgentOnlineWithoutStateObject(t *testing.T) {
//	cc := ClusterControllerImple{}
//	cc.Initialize()
//	agent1 := entities.LightningMonkeyAgent{
//		Id:            uuid.NewV4().String(),
//		ClusterId:     uuid.NewV4().String(),
//		Hostname:      "keepers-1",
//		IsDelete:      false,
//		HasETCDRole:   true,
//		HasMasterRole: false,
//		HasMinionRole: false,
//	}
//	err := cc.OnAgentChanged(agent1, false)
//	assert.Nil(t, err)
//	assert.True(t, len(cc.cache.etcd) == 0)
//}
//
//func Test_AgentOffline(t *testing.T) {
//	cc := ClusterControllerImple{}
//	cc.Initialize()
//	agent1 := entities.LightningMonkeyAgent{
//		Id:            uuid.NewV4().String(),
//		ClusterId:     uuid.NewV4().String(),
//		Hostname:      "keepers-1",
//		IsDelete:      false,
//		HasETCDRole:   true,
//		HasMasterRole: false,
//		HasMinionRole: false,
//		State: &entities.AgentState{
//			HasProvisionedETCD: true,
//		},
//	}
//	err := cc.OnAgentChanged(agent1, false)
//	assert.Nil(t, err)
//	assert.True(t, len(cc.cache.etcd) == 1)
//	agent1.State = nil
//	err = cc.OnAgentChanged(agent1, false)
//	assert.Nil(t, err)
//	assert.True(t, len(cc.cache.etcd) == 0)
//}
//
//func Test_DisposedCall(t *testing.T) {
//	cc := ClusterControllerImple{}
//	cc.Initialize()
//	agent1 := entities.LightningMonkeyAgent{
//		Id:            uuid.NewV4().String(),
//		ClusterId:     uuid.NewV4().String(),
//		Hostname:      "keepers-1",
//		IsDelete:      false,
//		HasETCDRole:   true,
//		HasMasterRole: false,
//		HasMinionRole: false,
//		State: &entities.AgentState{
//			HasProvisionedETCD: true,
//		},
//	}
//	isDisposed := 0
//	f := func() { isDisposed = 1 }
//	cc.SetCancellationFunc(f)
//	err := cc.OnAgentChanged(agent1, false)
//	assert.Nil(t, err)
//	assert.True(t, len(cc.cache.etcd) == 1)
//	cc.Dispose()
//	assert.Nil(t, cc.cache)
//	err = cc.OnAgentChanged(agent1, false)
//	assert.NotNil(t, err)
//	assert.True(t, isDisposed == 1)
//}
