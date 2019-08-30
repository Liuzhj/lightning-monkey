package test

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/mocks"
	"github.com/g0194776/lightningmonkey/pkg/cache"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/golang/mock/gomock"
	"github.com/googleapis/gnostic/compiler"
	"strings"
	"time"

	//"github.com/googleapis/gnostic/compiler"
	uuid "github.com/satori/go.uuid"
	assert "github.com/stretchr/testify/require"
	//"strings"
	"sync"
	"testing"
	//"time"
)

func Test_WithoutAnyLiveNodes(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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
	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	var currentAgent entities.LightningMonkeyAgent
	currentAgent.State = &entities.AgentState{}
	ac := cache.AgentCache{Mutex: &sync.Mutex{}}
	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.Nil(t, err)
	assert.NotNil(t, job)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_WithoutAnyExpectedETCDNodes(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	var currentAgent entities.LightningMonkeyAgent
	currentAgent.State = &entities.AgentState{}
	ac := cache.AgentCache{}

	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   false,
			HasMasterRole: true,
			HasMinionRole: false,
		},
	}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})

	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.Nil(t, err)
	assert.NotNil(t, job)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_LessThanExpectedETCDNodes(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	var currentAgent entities.LightningMonkeyAgent
	currentAgent.State = &entities.AgentState{}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		}}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})

	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.Nil(t, err)
	assert.NotNil(t, job)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_LessThanExpectedETCDNodes2(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	var currentAgent entities.LightningMonkeyAgent
	currentAgent.State = &entities.AgentState{}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		},
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers-2",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		}}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.Nil(t, err)
	assert.NotNil(t, job)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_CurrentAgentNotOnline(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)

	currentAgent := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		},
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers-2",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		},
		uuid.NewV4().String(): &currentAgent}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.NotNil(t, job)
	assert.NotNil(t, err)
	fmt.Printf("%#v\n", job)
	fmt.Printf("%#v\n", err)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_ProvisionedCountThanLessExpectedETCDNodeCount(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	currentAgent := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.1",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		},
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers-2",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		},
		uuid.NewV4().String(): &currentAgent}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_ProvisionedCountThanLessExpectedETCDNodeCount2(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	currentAgent := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.1",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
			State: &entities.AgentState{
				LastReportIP:       "127.0.0.1",
				HasProvisionedETCD: true,
				LastReportTime:     time.Now(),
			},
		},
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:        uuid.NewV4().String(),
			ClusterId: uuid.NewV4().String(),

			Hostname:      "keepers-2",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
		},
		uuid.NewV4().String(): &currentAgent}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_ProvisionedCountThanLessExpectedETCDNodeCount3(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	currentAgent := entities.LightningMonkeyAgent{
		Id:            uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "192.168.1.11",
			HasProvisionedETCD: false,
			LastReportTime:     time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:            uuid.NewV4().String(),
			ClusterId:     uuid.NewV4().String(),
			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
			State: &entities.AgentState{
				LastReportIP: "192.168.1.10",
			},
		},
		uuid.NewV4().String(): &currentAgent},
		map[string]*entities.LightningMonkeyAgent{},
		map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, currentAgent, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_ProvisionedCountByK8sMaster(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	gc := gomock.NewController(t)
	defer gc.Finish()

	currentAgent := entities.LightningMonkeyAgent{
		Id:            uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "192.168.1.11",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &entities.LightningMonkeyAgent{
			Id:            uuid.NewV4().String(),
			ClusterId:     uuid.NewV4().String(),
			Hostname:      "keepers",
			IsDelete:      false,
			HasETCDRole:   true,
			HasMasterRole: false,
			HasMinionRole: false,
			State: &entities.AgentState{
				LastReportIP:       "192.168.1.10",
				HasProvisionedETCD: true,
				LastReportTime:     time.Now(),
			},
		},
		uuid.NewV4().String(): &currentAgent},
		map[string]*entities.LightningMonkeyAgent{},
		map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	addr := ac.GetAgentsAddress(entities.AgentRole_ETCD, entities.AgentStatusFlag_Provisioned)
	assert.Equal(t, 2, len(addr))
}

func Test_GetETCDDeploymentJob(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.1",
			HasProvisionedETCD: false,
			LastReportTime:     time.Now(),
		},
	}
	agent2 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.2",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent3 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.3",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent1,
		uuid.NewV4().String(): &agent2,
		uuid.NewV4().String(): &agent3,
	}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, agent1, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_Deploy_ETCD)
	assert.True(t, job.Arguments != nil)
	assert.True(t, len(strings.Split(job.Arguments["addresses"], ",")) == 3)
	assert.True(t, compiler.StringArrayContainsValue(strings.Split(job.Arguments["addresses"], ","), agent1.State.LastReportIP))
	assert.True(t, compiler.StringArrayContainsValue(strings.Split(job.Arguments["addresses"], ","), agent2.State.LastReportIP))
	assert.True(t, compiler.StringArrayContainsValue(strings.Split(job.Arguments["addresses"], ","), agent3.State.LastReportIP))
}

func Test_WithoutAnyK8sMasterNodes(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.1",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent2 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.2",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent3 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.3",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent1,
		uuid.NewV4().String(): &agent2,
		uuid.NewV4().String(): &agent3,
	}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, agent1, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_GetK8sMasterDeploymentJob(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.1",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent2 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.2",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent3 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.3",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent4 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-4",
		IsDelete:      false,
		HasETCDRole:   false,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:                   "192.168.1.1",
			HasProvisionedMasterComponents: false,
			LastReportTime:                 time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent1,
		uuid.NewV4().String(): &agent2,
		uuid.NewV4().String(): &agent3,
	}, map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent4,
	}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, agent4, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_Deploy_Master)
}

func Test_WaitingAtLeastOneLiveK8sMaster(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs)

	agent1 := entities.LightningMonkeyAgent{
		Id:            uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.1",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent2 := entities.LightningMonkeyAgent{
		Id:            uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.2",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent3 := entities.LightningMonkeyAgent{
		Id:            uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.3",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent4 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-4",
		IsDelete:      false,
		HasETCDRole:   false,
		HasMasterRole: false,
		HasMinionRole: true,
		State: &entities.AgentState{
			LastReportIP:   "192.168.1.1",
			LastReportTime: time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent1,
		uuid.NewV4().String(): &agent2,
		uuid.NewV4().String(): &agent3,
	}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, agent4, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_NOP)
}

func Test_GetK8sMinionDeploymentJob(t *testing.T) {
	js := cache.ClusterJobSchedulerImple{}
	js.InitializeStrategies()

	cs := entities.LightningMonkeyClusterSettings{
		Name:              "demo_cluster",
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

	gc := gomock.NewController(t)
	defer gc.Finish()
	cc := mock_lm.NewMockClusterController(gc)
	cc.EXPECT().GetSettings().Return(cs).AnyTimes()

	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.1",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent2 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.2",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent3 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-3",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:       "127.0.0.3",
			HasProvisionedETCD: true,
			LastReportTime:     time.Now(),
		},
	}
	agent4 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-4",
		IsDelete:      false,
		HasETCDRole:   false,
		HasMasterRole: true,
		HasMinionRole: false,
		State: &entities.AgentState{
			LastReportIP:                   "192.168.1.1",
			LastReportTime:                 time.Now(),
			HasProvisionedMasterComponents: true,
		},
	}
	agent5 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-5",
		IsDelete:      false,
		HasETCDRole:   false,
		HasMasterRole: false,
		HasMinionRole: true,
		State: &entities.AgentState{
			LastReportIP:   "172.1.0.1",
			LastReportTime: time.Now(),
		},
	}
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent1,
		uuid.NewV4().String(): &agent2,
		uuid.NewV4().String(): &agent3,
	}, map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent4}, map[string]*entities.LightningMonkeyAgent{
		uuid.NewV4().String(): &agent5,
	}, map[string]*entities.LightningMonkeyAgent{})
	job, err := js.GetNextJob(cc, agent5, &ac)
	assert.NotNil(t, job)
	assert.Nil(t, err)
	fmt.Printf("%#v\n", job)
	assert.True(t, job.Name == entities.AgentJob_Deploy_Minion)
}

func Test_CacheOnline(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
}

func Test_CacheOnlineWithMultipleRoles(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 1)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
}

func Test_CacheOnlineWithMultipleRoles2(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: true,
		HasMinionRole: true,
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 1)
	assert.True(t, ac.GetKubernetesMinionCount() == 1)
}

func Test_CacheOffline(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
	ac.Offline(agent1)
	assert.True(t, ac.GetETCDCount() == 0)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
}

func Test_CacheOfflineWithMultipleRoles(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: true,
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 1)
	ac.Offline(agent1)
	assert.True(t, ac.GetETCDCount() == 0)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
}

func Test_CacheOfflineWithMultipleRoles2(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: true,
	}
	agent2 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   false,
		HasMasterRole: true,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	ac.Online(agent2)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 1)
	assert.True(t, ac.GetKubernetesMinionCount() == 1)
	ac.Offline(agent1)
	assert.True(t, ac.GetETCDCount() == 0)
	assert.True(t, ac.GetKubernetesMasterCount() == 1)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
}

func Test_CacheOfflineWithMultipleRoles3(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: true,
	}
	agent2 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-2",
		IsDelete:      false,
		HasETCDRole:   false,
		HasMasterRole: true,
		HasMinionRole: true,
	}
	ac.Online(agent1)
	ac.Online(agent2)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 1)
	assert.True(t, ac.GetKubernetesMinionCount() == 2)
	ac.Offline(agent1)
	assert.True(t, ac.GetETCDCount() == 0)
	assert.True(t, ac.GetKubernetesMasterCount() == 1)
	assert.True(t, ac.GetKubernetesMinionCount() == 1)
}

func Test_DulplicatedCacheOnline(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	ac.Online(agent1)
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
}

func Test_DulplicatedCacheOffline(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	ac.Online(agent1)
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
	ac.Offline(agent1)
	ac.Offline(agent1)
	ac.Offline(agent1)
	assert.True(t, ac.GetETCDCount() == 0)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
}

func Test_GetTotalCountWithSpecifiedRole(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
	assert.True(t, ac.GetTotalCountByRole(entities.AgentRole_ETCD) == 1)
	assert.True(t, ac.GetTotalCountByRole(entities.AgentRole_Minion) == 0)
}

func Test_GetTotalPrivisionedCountWithSpecifiedRole(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
	assert.True(t, ac.GetTotalProvisionedCountByRole(entities.AgentRole_ETCD) == 0)
	assert.True(t, ac.GetTotalProvisionedCountByRole(entities.AgentRole_Minion) == 0)
}

func Test_GetTotalPrivisionedCountWithSpecifiedRole2(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State:         &entities.AgentState{},
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
	assert.True(t, ac.GetTotalProvisionedCountByRole(entities.AgentRole_ETCD) == 0)
	assert.True(t, ac.GetTotalProvisionedCountByRole(entities.AgentRole_Minion) == 0)
}

func Test_GetTotalPrivisionedCountWithSpecifiedRole3(t *testing.T) {
	ac := cache.AgentCache{}
	ac.InitializeWithValues(map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{}, map[string]*entities.LightningMonkeyAgent{})
	agent1 := entities.LightningMonkeyAgent{
		Id:        uuid.NewV4().String(),
		ClusterId: uuid.NewV4().String(),

		Hostname:      "keepers-1",
		IsDelete:      false,
		HasETCDRole:   true,
		HasMasterRole: false,
		HasMinionRole: false,
		State: &entities.AgentState{
			HasProvisionedETCD: true,
		},
	}
	ac.Online(agent1)
	assert.True(t, ac.GetETCDCount() == 1)
	assert.True(t, ac.GetKubernetesMasterCount() == 0)
	assert.True(t, ac.GetKubernetesMinionCount() == 0)
	assert.True(t, ac.GetTotalProvisionedCountByRole(entities.AgentRole_ETCD) == 1)
	assert.True(t, ac.GetTotalProvisionedCountByRole(entities.AgentRole_Minion) == 0)
}
