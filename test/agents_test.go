package test

import (
	"context"
	"errors"
	"fmt"
	mock_lm "github.com/g0194776/lightningmonkey/mocks"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	assert "github.com/stretchr/testify/require"
	"go.etcd.io/etcd/clientv3"
	"strings"
	"testing"
	"time"
)

func Test_Successfully_Register_NewAgent(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	retValue := entities.LightningMonkeyClusterSettings{
		Id:                clusterId,
		CreateTime:        time.Now(),
		Name:              uuid.NewV4().String(),
		ExpectedETCDCount: 1,
		ServiceCIDR:       "10.254.1.1/16",
		KubernetesVersion: "1.12.5",
		PodNetworkCIDR:    "13.13.1.1/16",
		SecurityToken:     "",
		ServiceDNSDomain:  "cluster.local",
		NetworkStack: &entities.NetworkStackSettings{
			Type:       "kuberouter",
			Attributes: map[string]string{},
		},
	}
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterReady).AnyTimes()
	cc.EXPECT().GetSettings().Return(retValue)
	cc.EXPECT().GetCertificates().Return(entities.LightningMonkeyCertificateCollection{})

	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)
	cm.EXPECT().GetAgentFromETCD(clusterId, "").Return(nil, nil)

	sd := mock_lm.NewMockLightningMonkeyStorageDriver(gc)
	sd.EXPECT().GetRequestTimeoutDuration().Return(time.Second).AnyTimes()
	sd.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
		subKeys := strings.FieldsFunc(key, func(c rune) bool {
			return c == '/'
		})
		///correct formate: /lightning-monkey/clusters/XXXXXXXXXX/agents/XXXXXXXXXX/state
		assert.True(t, subKeys[0] == "lightning-monkey")
		assert.True(t, subKeys[1] == "clusters")
		assert.True(t, subKeys[len(subKeys)-1] == "settings")
		assert.True(t, subKeys[len(subKeys)-3] == "agents")
		return nil, nil
	}).Return(nil, nil)
	sd.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Do(func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
		subKeys := strings.FieldsFunc(key, func(c rune) bool {
			return c == '/'
		})
		///correct formate: /lightning-monkey/clusters/XXXXXXXXXX/agents/XXXXXXXXXX/state
		assert.True(t, subKeys[0] == "lightning-monkey")
		assert.True(t, subKeys[1] == "clusters")
		assert.True(t, subKeys[len(subKeys)-1] == "state")
		assert.True(t, subKeys[len(subKeys)-3] == "agents")
		return nil, nil
	}).Return(nil, nil)
	sd.EXPECT().NewLease().Return(&FakeETCDLease{})
	common.StorageDriver = sd

	certManager := mock_lm.NewMockCertificateManager(gc)
	gcm := &certs.GeneratedCertsMap{}
	gcm.InitializeData(map[string]string{
		"admin.conf": "sdfsdfsdf",
	})
	certManager.EXPECT().GenerateAdminKubeConfig("10.10.10.10", gomock.Any()).Return(gcm, nil)
	common.CertManager = certManager

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		ClusterId:     clusterId,
		Hostname:      uuid.NewV4().String(),
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	settings, agentId, _, leaseId, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}
	assert.True(t, settings != nil)
	assert.True(t, agentId != "")
	assert.True(t, leaseId == 100 /*faked value in the ETCD lease*/)
	assert.Nil(t, err)
}

func Test_Dulplicated_Register_ExistedAgent(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	agentId := uuid.NewV4().String()
	hostname := uuid.NewV4().String()
	retValue := entities.LightningMonkeyClusterSettings{
		Id:                clusterId,
		CreateTime:        time.Now(),
		Name:              uuid.NewV4().String(),
		ExpectedETCDCount: 1,
		ServiceCIDR:       "10.254.1.1/16",
		KubernetesVersion: "1.12.5",
		PodNetworkCIDR:    "13.13.1.1/16",
		SecurityToken:     "",
		ServiceDNSDomain:  "cluster.local",
		NetworkStack: &entities.NetworkStackSettings{
			Type:       "kuberouter",
			Attributes: map[string]string{},
		},
	}
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterReady).AnyTimes()
	cc.EXPECT().GetSettings().Return(retValue)
	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)

	preAgent := entities.LightningMonkeyAgent{
		Id:            agentId,
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}

	cm.EXPECT().GetAgentFromETCD(clusterId, agentId).Return(&preAgent, nil)

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		Id:            agentId,
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	settings, agentId, _, leaseId, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}
	assert.True(t, settings != nil)
	assert.True(t, agentId == agentId)
	assert.True(t, leaseId == -1)
	assert.Nil(t, err)
}

func Test_Failed_Register_ExistedAgent_DirtyOldAgentData(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	agentId := uuid.NewV4().String()
	hostname := uuid.NewV4().String()
	retValue := entities.LightningMonkeyClusterSettings{
		Id:                clusterId,
		CreateTime:        time.Now(),
		Name:              uuid.NewV4().String(),
		ExpectedETCDCount: 1,
		ServiceCIDR:       "10.254.1.1/16",
		KubernetesVersion: "1.12.5",
		PodNetworkCIDR:    "13.13.1.1/16",
		SecurityToken:     "",
		ServiceDNSDomain:  "cluster.local",
		NetworkStack: &entities.NetworkStackSettings{
			Type:       "kuberouter",
			Attributes: map[string]string{},
		},
	}
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterReady).AnyTimes()
	cc.EXPECT().GetSettings().Return(retValue)
	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)

	preAgent := entities.LightningMonkeyAgent{
		//Id:            agentId, <--dirty data, without any ID.
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}

	cm.EXPECT().GetAgentFromETCD(clusterId, agentId).Return(&preAgent, nil)

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		Id:            agentId,
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	_, _, _, _, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "dirty data"))
}

func Test_Failed_Register_ExistedAgent_DifferentLastReportIP(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	agentId := uuid.NewV4().String()
	hostname := uuid.NewV4().String()
	retValue := entities.LightningMonkeyClusterSettings{
		Id:                clusterId,
		CreateTime:        time.Now(),
		Name:              uuid.NewV4().String(),
		ExpectedETCDCount: 1,
		ServiceCIDR:       "10.254.1.1/16",
		KubernetesVersion: "1.12.5",
		PodNetworkCIDR:    "13.13.1.1/16",
		SecurityToken:     "",
		ServiceDNSDomain:  "cluster.local",
		NetworkStack: &entities.NetworkStackSettings{
			Type:       "kuberouter",
			Attributes: map[string]string{},
		},
	}
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterReady).AnyTimes()
	cc.EXPECT().GetSettings().Return(retValue)
	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)

	preAgent := entities.LightningMonkeyAgent{
		Id:            agentId,
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.11", //<--dirty data, mismatched with original client IP.
		},
	}

	cm.EXPECT().GetAgentFromETCD(clusterId, agentId).Return(&preAgent, nil)

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		Id:            agentId,
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	_, _, _, _, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "Duplicated agent registering with different client IP!"))
}

func Test_Failed_Register_ExistedAgent_RemovedOldAgent(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	agentId := uuid.NewV4().String()
	hostname := uuid.NewV4().String()
	retValue := entities.LightningMonkeyClusterSettings{
		Id:                clusterId,
		CreateTime:        time.Now(),
		Name:              uuid.NewV4().String(),
		ExpectedETCDCount: 1,
		ServiceCIDR:       "10.254.1.1/16",
		KubernetesVersion: "1.12.5",
		PodNetworkCIDR:    "13.13.1.1/16",
		SecurityToken:     "",
		ServiceDNSDomain:  "cluster.local",
		NetworkStack: &entities.NetworkStackSettings{
			Type:       "kuberouter",
			Attributes: map[string]string{},
		},
	}
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterReady).AnyTimes()
	cc.EXPECT().GetSettings().Return(retValue)
	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)

	preAgent := entities.LightningMonkeyAgent{
		Id:            agentId,
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		IsDelete:      true, //<--old agent indicated that it already was deleted by control center.
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}

	cm.EXPECT().GetAgentFromETCD(clusterId, agentId).Return(&preAgent, nil)

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		Id:            agentId,
		ClusterId:     clusterId,
		Hostname:      hostname,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	_, _, _, _, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "Target registered agent has been deleted, Please do not reuse it again!"))
}

func Test_Register_NewAgent_FailedGenerateCertificate(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	retValue := entities.LightningMonkeyClusterSettings{
		Id:                clusterId,
		CreateTime:        time.Now(),
		Name:              uuid.NewV4().String(),
		ExpectedETCDCount: 1,
		ServiceCIDR:       "10.254.1.1/16",
		KubernetesVersion: "1.12.5",
		PodNetworkCIDR:    "13.13.1.1/16",
		SecurityToken:     "",
		ServiceDNSDomain:  "cluster.local",
		NetworkStack: &entities.NetworkStackSettings{
			Type:       "kuberouter",
			Attributes: map[string]string{},
		},
	}
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterReady).AnyTimes()
	cc.EXPECT().GetSettings().Return(retValue)
	cc.EXPECT().GetCertificates().Return(entities.LightningMonkeyCertificateCollection{})

	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)
	cm.EXPECT().GetAgentFromETCD(clusterId, "").Return(nil, nil)

	certManager := mock_lm.NewMockCertificateManager(gc)
	gcm := &certs.GeneratedCertsMap{}
	gcm.InitializeData(map[string]string{
		"admin.conf": "sdfsdfsdf",
	})
	certManager.EXPECT().GenerateAdminKubeConfig("10.10.10.10", gomock.Any()).Return(nil, errors.New("failed to generate certificate!"))
	common.CertManager = certManager

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		ClusterId:     clusterId,
		Hostname:      uuid.NewV4().String(),
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	_, _, _, _, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to generate certificate!"))
}

func Test_Register_NewAgent_AdminCertNotFound(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	retValue := entities.LightningMonkeyClusterSettings{
		Id:                clusterId,
		CreateTime:        time.Now(),
		Name:              uuid.NewV4().String(),
		ExpectedETCDCount: 1,
		ServiceCIDR:       "10.254.1.1/16",
		KubernetesVersion: "1.12.5",
		PodNetworkCIDR:    "13.13.1.1/16",
		SecurityToken:     "",
		ServiceDNSDomain:  "cluster.local",
		NetworkStack: &entities.NetworkStackSettings{
			Type:       "kuberouter",
			Attributes: map[string]string{},
		},
	}
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterReady).AnyTimes()
	cc.EXPECT().GetSettings().Return(retValue)
	cc.EXPECT().GetCertificates().Return(entities.LightningMonkeyCertificateCollection{})

	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)
	cm.EXPECT().GetAgentFromETCD(clusterId, "").Return(nil, nil)

	certManager := mock_lm.NewMockCertificateManager(gc)
	gcm := &certs.GeneratedCertsMap{}
	gcm.InitializeData(map[string]string{
		"admin2.conf": "sdfsdfsdf",
	})
	certManager.EXPECT().GenerateAdminKubeConfig("10.10.10.10", gomock.Any()).Return(gcm, nil)
	common.CertManager = certManager

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		ClusterId:     clusterId,
		Hostname:      uuid.NewV4().String(),
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	_, _, _, _, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "Failed to generate kube admin configuration file, generated config file not found!"))
}

func Test_Register_NewAgent_RemovedCluster(t *testing.T) {
	gc := gomock.NewController(t)
	defer gc.Finish()

	clusterId := uuid.NewV4().String()
	cm := mock_lm.NewMockClusterManagerInterface(gc)
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetStatus().Return(entities.ClusterDeleted).AnyTimes()
	cc.EXPECT().GetClusterId().Return(clusterId)
	cm.EXPECT().GetClusterById(clusterId).Return(cc, nil)

	common.ClusterManager = cm
	agent := entities.LightningMonkeyAgent{
		ClusterId:     clusterId,
		Hostname:      uuid.NewV4().String(),
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP: "10.10.10.10",
		},
	}
	_, _, _, _, err := managers.RegisterAgent(&agent)
	if err != nil {
		t.Logf("%#v", err)
	}

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("Target cluster: %s had been deleted.", clusterId)))
}
