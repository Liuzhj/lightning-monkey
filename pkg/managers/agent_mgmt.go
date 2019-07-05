package managers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"strings"
	"time"
)

func RegisterAgent(agent *entities.LightningMonkeyAgent) (*entities.LightningMonkeyClusterSettings, string, int64, error) {
	if agent.ClusterId == "" {
		return nil, "", -1, errors.New("HTTP body field \"cluster_id\" is required for registering agent.")
	}
	if agent.Hostname == "" {
		return nil, "", -1, errors.New("HTTP body field \"hostname\" is required for registering agent.")
	}
	cluster, err := common.ClusterManager.GetClusterById(agent.ClusterId)
	if err != nil {
		return nil, "", -1, fmt.Errorf("Failed to retrieve cluster information from database, error: %s", err.Error())
	}
	if cluster.GetStatus() == entities.ClusterDeleted {
		return nil, "", -1, fmt.Errorf("Target cluster: %s had been deleted.", cluster.GetClusterId())
	}
	settings := cluster.GetSettings()
	if cluster.GetStatus() == entities.ClusterBlockedAgentRegistering {
		return nil, "", -1, fmt.Errorf("Target cluster: %s has been blocked agent registering, try it later.", settings.Name)
	}
	preAgent, err := common.ClusterManager.GetAgentFromETCD(agent.ClusterId, agent.Id)
	if err != nil {
		return nil, "", -1, fmt.Errorf("Failed to retrieve agent information from database, error: %s", err.Error())
	}
	if preAgent != nil {
		if preAgent.Id == "" {
			return nil, "", -1, errors.New("Old agent has one or more fields contains dirty data.")
		}
		if agent.ClusterId != preAgent.ClusterId {
			return nil, "", -1, errors.New("Duplicated agent registering with wrong cluster!")
		}
		if strings.ToLower(agent.Hostname) != strings.ToLower(preAgent.Hostname) {
			return nil, "", -1, errors.New("Duplicated agent registering with different hostname!")
		}
		if agent.State != nil && preAgent.State != nil {
			if agent.State.LastReportIP != preAgent.State.LastReportIP {
				return nil, "", -1, errors.New("Duplicated agent registering with different client IP!")
			}
		}
		if preAgent.IsDelete {
			return nil, "", -1, errors.New("Target registered agent has been deleted, Please do not reuse it again!")
		}
		//duplicated registering.
		return &settings, preAgent.Id, -1, nil
	}
	//generate admin config for master role agent.
	if agent.HasMasterRole {
		certMap := cluster.GetCertificates()
		adminKubeCert, err := common.CertManager.GenerateAdminKubeConfig(agent.State.LastReportIP, certMap)
		if err != nil {
			return nil, "", -1, fmt.Errorf("Failed to generate kube admin configuration file, error: %s", err.Error())
		}
		res := adminKubeCert.GetResources()
		if res == nil || len(res) == 0 {
			return nil, "", -1, errors.New("Failed to generate kube admin configuration file, generated config file not found!")
		}
		agent.AdminCertificate = adminKubeCert.GetResources()["admin.conf"]
		if agent.AdminCertificate == "" {
			return nil, "", -1, errors.New("Failed to generate kube admin configuration file, generated config file not found!")
		}
	}
	agent.Id = uuid.NewV4().String()
	agent.State.LastReportTime = time.Now()
	leaseId, err := saveAgent(agent)
	if err != nil {
		return nil, "", -1, fmt.Errorf("Failed to save registered agent to storage driver, error: %s", err.Error())
	}
	return &settings, agent.Id, leaseId, nil
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
	job, err := cluster.GetNextJob(*agent)
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
	if v, isOK := status.Items[entities.AgentRole_ETCD]; isOK {
		state.HasProvisionedETCD = v.HasProvisioned
		state.Reason = v.Reason //TODO: different component's reason should explicitly break off.
	} else {
		state.HasProvisionedETCD = false
	}
	//detect Kubernetes Master deployment status.
	if v, isOK := status.Items[entities.AgentRole_Master]; isOK {
		state.HasProvisionedMasterComponents = v.HasProvisioned
		state.Reason = v.Reason //TODO: different component's reason should explicitly break off.
	} else {
		state.HasProvisionedMasterComponents = false
	}
	//detect Kubernetes Minion deployment status.
	if v, isOK := status.Items[entities.AgentRole_Minion]; isOK {
		state.HasProvisionedMinion = v.HasProvisioned
		state.Reason = v.Reason //TODO: different component's reason should explicitly break off.
	} else {
		state.HasProvisionedMinion = false
	}
	return saveAgentState(clusterId, agentId, status.LeaseId, &state)
}

func saveAgent(agent *entities.LightningMonkeyAgent) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.StorageDriver.GetRequestTimeoutDuration())
	defer cancel()
	//STEP 1, save agent's settings.
	path := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/settings", agent.ClusterId, agent.Id)
	data, err := json.Marshal(agent)
	if err != nil {
		return -1, err
	}
	_, err = common.StorageDriver.Put(ctx, path, string(data))
	if err != nil {
		return -1, err
	}
	//STEP 2, save agent's state with TTL.
	return saveAgentStateWithTTL(agent.ClusterId, agent.Id, agent.State)
}

func saveAgentStateWithTTL(clusterId string, agentId string, state *entities.AgentState) (int64, error) {
	leaseId, err := newETCDLease()
	if err != nil {
		return -1, err
	}
	path := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/state", clusterId, agentId)
	data, err := json.Marshal(state)
	if err != nil {
		return -1, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), common.StorageDriver.GetRequestTimeoutDuration())
	defer cancel()
	_, err = common.StorageDriver.Put(ctx, path, string(data), clientv3.WithLease(clientv3.LeaseID(leaseId)))
	if err != nil {
		return -1, err
	}
	return leaseId, nil
}

func newETCDLease() (int64, error) {
	lease := common.StorageDriver.NewLease()
	grantRsp, err := lease.Grant(context.TODO(), 15)
	if err != nil {
		return -1, fmt.Errorf("Could not grant a new lease to remote storage driver, error: %s", err.Error())
	}
	return int64(grantRsp.ID), nil
}

func saveAgentState(clusterId string, agentId string, leaseId int64, state *entities.AgentState) (int64, error) {
	var err error
	needRegenerateLease := leaseId == -1
	//STEP 1, renew/regenerate ETCD key lease.
	if needRegenerateLease {
		leaseId, err = newETCDLease()
		if err != nil {
			return -1, err
		}
		logrus.Infof("Agent %s has triggered reconnection procedure, state lease will renew one.", agentId)
	} else {
		lease := common.StorageDriver.NewLease()
		_, err := lease.KeepAliveOnce(context.TODO(), clientv3.LeaseID(leaseId))
		if err != nil {
			return -1, fmt.Errorf("Failed to renew lease to remote storage driver, error: %s", err.Error())
		}
	}
	//STEP 2, update state.
	path := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/state", clusterId, agentId)
	data, err := json.Marshal(state)
	if err != nil {
		return leaseId, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), common.StorageDriver.GetRequestTimeoutDuration())
	defer cancel()
	_, err = common.StorageDriver.Put(ctx, path, string(data), clientv3.WithLease(clientv3.LeaseID(leaseId)))
	return leaseId, err
}
