package managers

import (
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/globalsign/mgo/bson"
	"strings"
	"time"
)

func RegisterAgent(agent *entities.Agent) (*entities.Cluster, error) {
	if agent.ClusterId == nil {
		return nil, errors.New("HTTP body field \"cluster_id\" is required for registering agent.")
	}
	if agent.MetadataId == "" {
		return nil, errors.New("HTTP body field \"metadata_id\" is required for registering agent.")
	}
	if agent.Hostname == "" {
		return nil, errors.New("HTTP body field \"hostname\" is required for registering agent.")
	}
	cluster, err := common.StorageDriver.GetCluster(agent.ClusterId.Hex())
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve cluster information from database, error: %s", err.Error())
	}
	if cluster.Status == entities.ClusterDeleted {
		return nil, fmt.Errorf("Target cluster: %s had been deleted.", cluster.Name)
	}
	if cluster.Status == entities.ClusterBlockedAgentRegistering {
		return nil, fmt.Errorf("Target cluster: %s has been blocked agent registering, try it later.", cluster.Name)
	}
	preAgent, err := common.StorageDriver.GetAgentByMetadataId(agent.MetadataId)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve agent information from database, error: %s", err.Error())
	}
	if preAgent != nil {
		if agent.ClusterId.Hex() != preAgent.ClusterId.Hex() {
			return nil, errors.New("Duplicated agent registering with wrong cluster!")
		}
		if strings.ToLower(agent.Hostname) != strings.ToLower(preAgent.Hostname) {
			return nil, errors.New("Duplicated agent registering with different hostname!")
		}
		if agent.LastReportIP != preAgent.LastReportIP {
			return nil, errors.New("Duplicated agent registering with different client IP!")
		}
		if preAgent.IsDelete {
			return nil, errors.New("Target registered agent has been deleted, Please do not reuse it again!")
		}
		//duplicated registering.
		return cluster, nil
	}
	agentId := bson.NewObjectId()
	agent.Id = &agentId
	agent.LastReportTime = time.Now()
	err = common.StorageDriver.SaveAgent(agent)
	if err != nil {
		return nil, fmt.Errorf("Failed to save registered agent to database, error: %s", err.Error())
	}
	return cluster, nil
}

func QueryAgentNextWorkItem(metadataId string) (*entities.AgentJob, error) {
	agent, err := common.StorageDriver.GetAgentByMetadataId(metadataId)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve agent from database, error: %s", err.Error())
	}
	if agent == nil {
		return nil, errors.New("Current agent has not registered to Master.")
	}
	cluster, err := common.StorageDriver.GetCluster(agent.ClusterId.Hex())
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve the cluster information which current agent belongs to, error: %s", err.Error())
	}
	if cluster == nil {
		return nil, errors.New("Agent's cluster information had been deleted(Maybe), not found.")
	}
	strategy := common.ClusterStatementController.GetClusterStrategy(cluster.Id.Hex())
	if strategy == nil {
		//not load into memory yet.
		return nil, nil
	}
	canDeployETCD := strategy.CanDeployETCD()
	canDeployMaster := strategy.CanDeployMasterComponents()
	canDeployMinion := strategy.CanDeployMinion()
	//1st, deploy ETCD components.
	if canDeployETCD == entities.ConditionNotConfirmed {
		return &entities.AgentJob{Name: entities.AgentJob_NOP, Reason: "Wait, All of agents of ETCD role are not ready yet."}, nil
	}
	if agent.HasETCDRole && !agent.HasProvisionedETCD && canDeployETCD == entities.ConditionConfirmed {
		return &entities.AgentJob{Name: entities.AgentJob_Deploy_ETCD, Arguments: map[string]string{"addresses": strings.Join(strategy.GetETCDNodeAddresses(), ",")}}, nil
	}
	//2ec, deploy Master components.
	if canDeployMaster == entities.ConditionNotConfirmed {
		return &entities.AgentJob{Name: entities.AgentJob_NOP, Reason: "Wait, All of agents of Kubernetes master role are not ready yet."}, nil
	}
	if agent.HasMasterRole && !agent.HasProvisionedMasterComponents && canDeployMaster == entities.ConditionConfirmed {
		return &entities.AgentJob{Name: entities.AgentJob_Deploy_Master}, nil
	}
	//last, deploy Minion components.
	if agent.HasMinionRole && !agent.HasProvisionedMinion && canDeployMinion == entities.ConditionConfirmed {
		return &entities.AgentJob{Name: entities.AgentJob_Deploy_Minion}, nil
	}
	return &entities.AgentJob{Name: entities.AgentJob_NOP, Reason: "Wait, no any operations should perform."}, nil
}

func AgentReportStatus(metadataId string, status entities.AgentStatus) error {
	agent, err := common.StorageDriver.GetAgentByMetadataId(metadataId)
	if err != nil {
		return fmt.Errorf("Failed to retrieve agent from database, error: %s", err.Error())
	}
	if agent == nil {
		return errors.New("Current agent has not registered to Master.")
	}
	cluster, err := common.StorageDriver.GetCluster(agent.ClusterId.Hex())
	if err != nil {
		return fmt.Errorf("Failed to retrieve the cluster information which current agent belongs to, error: %s", err.Error())
	}
	if cluster == nil {
		return errors.New("Agent's cluster information had been deleted(Maybe), not found.")
	}
	agent.LastReportTime = time.Now()
	agent.LastReportStatus = status.Status
	agent.Reason = status.Reason
	if status.Status == entities.AgentStatus_Provision_Succeed && status.ReportTarget == entities.AgentJob_Deploy_ETCD {
		agent.HasProvisionedETCD = true
		agent.ETCDProvisionTime = time.Now()
	}
	if status.Status == entities.AgentStatus_Provision_Succeed && status.ReportTarget == entities.AgentJob_Deploy_Master {
		agent.HasProvisionedMasterComponents = true
		agent.MasterComponentsProvisionTime = time.Now()
	}
	if status.Status == entities.AgentStatus_Provision_Succeed && status.ReportTarget == entities.AgentJob_Deploy_Minion {
		agent.HasProvisionedMinion = true
		agent.MinionProvisionTime = time.Now()
	}
	//strategy := common.ClusterStatementController.GetClusterStrategy(agent.ClusterId.Hex())
	//if strategy != nil {
	//	strategy.UpdateAgentProvisionStatus()
	//}
	return common.StorageDriver.SaveAgent(agent)
}
