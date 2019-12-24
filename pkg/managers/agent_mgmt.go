package managers

import (
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func RegisterAgent(agent *entities.LightningMonkeyAgent) (*entities.LightningMonkeyClusterSettings, string, string, int64, error) {
	if agent.ClusterId == "" {
		return nil, "", "", -1, errors.New("HTTP body field \"cluster_id\" is required for registering agent.")
	}
	if agent.Hostname == "" {
		return nil, "", "", -1, errors.New("HTTP body field \"hostname\" is required for registering agent.")
	}
	cluster, err := common.ClusterManager.GetClusterById(agent.ClusterId)
	if err != nil {
		return nil, "", "", -1, fmt.Errorf("Failed to retrieve cluster information from database, error: %s", err.Error())
	}
	if cluster.GetStatus() == entities.ClusterDeleted {
		return nil, "", "", -1, fmt.Errorf("Target cluster: %s had been deleted.", cluster.GetClusterId())
	}
	settings := cluster.GetSettings()
	if cluster.GetStatus() == entities.ClusterBlockedAgentRegistering {
		return nil, "", "", -1, fmt.Errorf("Target cluster: %s has been blocked agent registering, try it later.", settings.Name)
	}
	preAgent, err := common.ClusterManager.GetAgentFromETCD(agent.ClusterId, agent.Id)
	if err != nil {
		return nil, "", "", -1, fmt.Errorf("Failed to retrieve agent information from database, error: %s", err.Error())
	}
	if preAgent != nil {
		if preAgent.Id == "" {
			return nil, "", "", -1, errors.New("Old agent has one or more fields contains dirty data.")
		}
		if strings.ToLower(agent.Hostname) != strings.ToLower(preAgent.Hostname) {
			return nil, "", "", -1, errors.New("Duplicated agent registering with different hostname!")
		}
		if agent.State != nil && preAgent.State != nil {
			if agent.State.LastReportIP != preAgent.State.LastReportIP {
				return nil, "", "", -1, errors.New("Duplicated agent registering with different client IP!")
			}
		}
		if preAgent.IsDelete {
			return nil, "", "", -1, errors.New("Target registered agent has been deleted, Please do not reuse it again!")
		}
		//duplicated registering.
		return &settings, preAgent.Id, preAgent.ClusterId, -1, nil
	}
	//generate admin config for master role agent.
	if agent.HasMasterRole {
		certMap := cluster.GetCertificates()
		logrus.Infof("Cluster: %s certs count: %d", agent.ClusterId, len(certMap))
		adminKubeCert, err := common.CertManager.GenerateAdminKubeConfig(agent.State.LastReportIP, certMap)
		if err != nil {
			return nil, "", "", -1, fmt.Errorf("Failed to generate kube admin configuration file, error: %s", err.Error())
		}
		res := adminKubeCert.GetResources()
		if res == nil || len(res) == 0 {
			return nil, "", "", -1, errors.New("Failed to generate kube admin configuration file, generated config file not found!")
		}
		agent.AdminCertificate = adminKubeCert.GetResources()["admin.conf"]
		if agent.AdminCertificate == "" {
			return nil, "", "", -1, errors.New("Failed to generate kube admin configuration file, generated config file not found!")
		}
	}
	if agent.Id == "" {
		agent.Id = uuid.NewV4().String()
	}
	agent.State.LastReportTime = time.Now()
	leaseId, err := common.SaveAgent(agent)
	if err != nil {
		return nil, "", "", -1, fmt.Errorf("Failed to save registered agent to storage driver, error: %s", err.Error())
	}
	return &settings, agent.Id, settings.Id, leaseId, nil
}

func QueryAgentNextWorkItem(clusterId, agentId string) (*entities.AgentJob, error) {
	cluster, err := common.ClusterManager.GetClusterById(clusterId)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve cluster information from cache, error: %s", err.Error())
	}
	if cluster == nil {
		return nil, fmt.Errorf("Cluster: %s not found!", clusterId)
	}
	agent, err := cluster.GetCachedAgent(agentId)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve agent information from cache, cluster-id: %s, agent-id: %s, error: %s", clusterId, agentId, err.Error())
	}
	if agent == nil {
		return nil, fmt.Errorf("Agent: %s not found!", agentId)
	}
	oldDeploymentPhase := agent.DeploymentPhase
	job, err := cluster.GetNextJob(*agent, func(i int) {
		agent.DeploymentPhase = i
	})
	if agent.DeploymentPhase > oldDeploymentPhase {
		internalErr := common.SaveAgentSettingsOnly(agent)
		if internalErr != nil {
			logrus.Errorf("Failed to save agent %s settings which triggered by deployment phase updating(%d -> %d), error: %s", agentId, oldDeploymentPhase, agent.DeploymentPhase, internalErr.Error())
		}
	}
	return &job, err
}

func AgentReportStatus(clusterId, agentId string, status entities.LightningMonkeyAgentReportStatus) (int64, error) {
	cluster, err := common.ClusterManager.GetClusterById(clusterId)
	if err != nil {
		return -1, fmt.Errorf("Failed to retrieve cluster information from cache, error: %s", err.Error())
	}
	if cluster == nil {
		return -1, fmt.Errorf("Cluster: %s not found!", clusterId)
	}
	agent, err := cluster.GetCachedAgent(agentId)
	if err != nil {
		return -1, fmt.Errorf("Failed to retrieve agent information from cache, cluster-id: %s, agent-id: %s, error: %s", clusterId, agentId, err.Error())
	}
	//missed cache on L1, try retrieving it from L2 cache.
	if agent == nil {
		agent, err = common.ClusterManager.GetAgentFromETCD(clusterId, agentId)
		if err != nil {
			return -1, fmt.Errorf("Failed to retrieve agent information from L2 cache, cluster-id: %s, agent-id: %s, error: %s", clusterId, agentId, err.Error())
		}
		if agent == nil {
			return -1, fmt.Errorf("Agent: %s not found!", agentId)
		}
		//considered to regenerate it which currently held on client-side.
		status.LeaseId = -1
	}
	state := entities.AgentState{}
	state.LastReportIP = status.IP
	state.LastReportTime = time.Now()
	//detect ETCD deployment status.
	if v, isOK := status.Items[entities.AgentJob_Deploy_ETCD]; isOK {
		state.HasProvisionedETCD = v.HasProvisioned
		state.Reason = v.Reason //TODO: different component's reason should explicitly break off.
	} else {
		state.HasProvisionedETCD = false
	}
	//detect Kubernetes Master deployment status.
	if v, isOK := status.Items[entities.AgentJob_Deploy_Master]; isOK {
		state.HasProvisionedMasterComponents = v.HasProvisioned
		state.Reason = v.Reason //TODO: different component's reason should explicitly break off.
	} else {
		state.HasProvisionedMasterComponents = false
	}
	//detect HA deployment status.
	if v, isOK := status.Items[entities.AgentJob_Deploy_HA]; isOK {
		state.HasProvisionedHA = v.HasProvisioned
		state.Reason = v.Reason //TODO: different component's reason should explicitly break off.
	} else {
		state.HasProvisionedHA = false
	}
	//detect Kubernetes Minion deployment status.
	if v, isOK := status.Items[entities.AgentJob_Deploy_Minion]; isOK {
		state.HasProvisionedMinion = v.HasProvisioned
		state.Reason = v.Reason //TODO: different component's reason should explicitly break off.
	} else {
		state.HasProvisionedMinion = false
	}
	return common.SaveAgentStateOnly(clusterId, agentId, status.LeaseId, &state)
}
