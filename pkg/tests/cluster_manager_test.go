package tests

import (
	"github.com/g0194776/lightningmonkey/mocks"
	"github.com/g0194776/lightningmonkey/pkg/cache"
	"github.com/golang/mock/gomock"
	assert "github.com/stretchr/testify/require"
	"go.etcd.io/etcd/clientv3"
	"testing"
)

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
