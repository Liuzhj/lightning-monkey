package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"io/ioutil"
	"net/http"
	"time"
)

//TransferAgentToCluster allowed to transfer an agent to another one cluster.
func (cm *ClusterManager) TransferAgentToCluster(oldClusterId string, newClusterId string, agent *entities.LightningMonkeyAgent, isETCDRole, isMasterRole, isMinionRole, isHARole bool) error {
	//STEP 1, entirely remove all of OLD agent's data to remote ETCD,
	//it'll cause the data changing notification to the all of API Servers for cache cleaning.
	err := cm.RemoveAgentFromETCD(oldClusterId, agent.Id)
	if err != nil {
		return fmt.Errorf("Failed to entirely remove given agent(%s) from remote ETCD, error: %s", agent.Id, err.Error())
	}
	//STEP 2, make a call to the agent API for re-registering to the new cluster.
	err = changeAgentCluster(agent, oldClusterId, newClusterId, isETCDRole, isMasterRole, isMinionRole, isHARole)
	if err != nil {
		return fmt.Errorf("Failed to notify agent(%s) API to change the cluster-id and roles, error: %s", agent.Id, err.Error())
	}
	return nil
}

func changeAgentCluster(agent *entities.LightningMonkeyAgent, oldClusterId string, newClusterId string, isETCDRole, isMasterRole, isMinionRole, isHARole bool) error {
	gr := entities.ChangeClusterAndRolesRequest{
		OldClusterId: oldClusterId,
		NewClusterId: newClusterId,
		IsETCDRole:   isETCDRole,
		IsMasterRole: isMasterRole,
		IsMinionRole: isMinionRole,
		IsHARole:     isHARole,
	}
	data, err := json.Marshal(gr)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/registration/change", agent.State.LastReportIP, agent.ListenPort), bytes.NewReader(data))
	if err != nil {
		return err
	}
	client := http.Client{
		Timeout:   time.Second * 5,
		Transport: http.DefaultTransport,
	}
	rsp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote agent had returned non-Zero HTTP status code: %d", rsp.StatusCode)
	}
	data, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	var genericResponse entities.Response
	err = json.Unmarshal(data, &genericResponse)
	if err != nil {
		return err
	}
	if genericResponse.ErrorId != 0 {
		if genericResponse.Reason != "" {
			return fmt.Errorf("remote agent had returned an error(%d): %s", genericResponse.ErrorId, genericResponse.Reason)
		} else {
			return fmt.Errorf("remote agent had returned non-Zero biz error-id: %d", genericResponse.ErrorId)
		}
	}
	return nil
}

func (cm *ClusterManager) RemoveAgentFromETCD(clusterId string, agentId string) error {
	return nil
}
