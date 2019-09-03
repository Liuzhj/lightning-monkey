package main

import (
	"github.com/docker/engine-api/client"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var (
	CERTIFICATE_STORAGE_PATH = "/etc/kubernetes"
	RECOVERY_FILE_PATH       = "/opt/lightning-monkey/recovery"
	crashError               = errors.New("CRASH ERROR")
)

const (
	COMPONENT_DEPLOYMENT_EXTERNAL    = "external"
	COMPONENT_DEPLOYMENT_INTEGRATION = "integration"
)

type LightningMonkeyAgentReportStatus struct {
	Item entities.LightningMonkeyAgentReportStatusItem
	Key  string
}

type LightningMonkeyAgent struct {
	c                     chan LightningMonkeyAgentReportStatus
	currentJob            *entities.AgentJob
	statusLock            *sync.RWMutex
	recoveryLock          *sync.Mutex
	arg                   *AgentArgs
	dockerClient          *client.Client
	dockerImageManager    managers.DockerImageManager
	lastRegisteredTime    time.Time
	lastReportTime        time.Time
	hasRegistered         int32
	basicImages           *entities.DockerImageCollection
	masterSettings        map[string]string
	workQueue             chan *entities.AgentJob
	handlerFactory        *AgentJobHandlerFactory
	ItemsStatus           map[string]entities.LightningMonkeyAgentReportStatusItem
	expectedETCDNodeCount int
	rr                    *RecoveryRecord
}

type RecoveryRecord struct {
	HasInstalledMaster   bool      `json:"has_installed_master"`
	HasInstalledETCD     bool      `json:"has_installed_etcd"`
	HasInstalledMinion   bool      `json:"has_installed_minion"`
	InstallMasterTime    time.Time `json:"install_master_time"`
	InstallETCDTime      time.Time `json:"install_etcd_time"`
	InstallMinionTime    time.Time `json:"install_minion_time"`
	MasterDeploymentType string    `json:"master_deployment_type"`
	ETCDDeploymentType   string    `json:"etcd_deployment_type"`
	MinionDeploymentType string    `json:"minion_deployment_type"`
}
