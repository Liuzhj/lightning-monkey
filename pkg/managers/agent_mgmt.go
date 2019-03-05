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

func RegisterAgent(agent *entities.Agent) error {
	if agent.ClusterId == nil {
		return errors.New("HTTP body field \"cluster_id\" is required for registering agent.")
	}
	if agent.MetadataId == "" {
		return errors.New("HTTP body field \"metadata_id\" is required for registering agent.")
	}
	if agent.Hostname == "" {
		return errors.New("HTTP body field \"hostname\" is required for registering agent.")
	}
	if agent.Roles == nil || len(agent.Roles) == 0 {
		return errors.New("Current being register agent must have one legal role at least.")
	}
	cluster, err := common.StorageDriver.GetCluster(agent.ClusterId.Hex())
	if err != nil {
		return fmt.Errorf("Failed to retrieve cluster information from database, error: %s", err.Error())
	}
	if cluster.Status == entities.ClusterDeleted {
		return fmt.Errorf("Target cluster: %s had been deleted.", cluster.Name)
	}
	if cluster.Status == entities.ClusterBlockedAgentRegistering {
		return fmt.Errorf("Target cluster: %s has been blocked agent registering, try it later.", cluster.Name)
	}
	preAgent, err := common.StorageDriver.GetAgentByMetadataId(agent.MetadataId)
	if err != nil {
		return fmt.Errorf("Failed to retrieve agent information from database, error: %s", err.Error())
	}
	if preAgent != nil {
		if agent.ClusterId.Hex() != preAgent.ClusterId.Hex() {
			return errors.New("Duplicated agent registering with wrong cluster!")
		}
		if strings.ToLower(agent.Hostname) != strings.ToLower(preAgent.Hostname) {
			return errors.New("Duplicated agent registering with different hostname!")
		}
		if agent.LastReportIP != preAgent.LastReportIP {
			return errors.New("Duplicated agent registering with different client IP!")
		}
		if preAgent.IsDelete {
			return errors.New("Target registered agent has been deleted, Please do not reuse it again!")
		}
		//duplicated registering.
		return nil
	}
	agentId := bson.NewObjectId()
	agent.Id = &agentId
	agent.LastReportStatus = "NEW"
	agent.LastReportTime = time.Now()
	err = common.StorageDriver.SaveAgent(agent)
	if err != nil {
		return fmt.Errorf("Failed to save registered agent to database, error: %s", err.Error())
	}
	return nil
}

func GetAgentById(metadataId string) (*entities.Agent, error) {
	return nil, nil
}
