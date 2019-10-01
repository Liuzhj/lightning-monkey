package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/monitors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

type MetricsServerAddStaticRouteStrategy struct {
}

func (js *MetricsServerAddStaticRouteStrategy) GetStrategyName() string {
	return "Metrics-Server Static Routing"
}

func (js *MetricsServerAddStaticRouteStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	if !agent.HasMasterRole || !agent.State.HasProvisionedMasterComponents {
		return entities.ConditionInapplicable, "", nil, nil
	}
	cs := cc.GetSettings()
	if cs.ExtensionalDeployments == nil || len(cs.ExtensionalDeployments) == 0 {
		return entities.ConditionInapplicable, "", nil, nil
	}
	if _, isOK := cs.ExtensionalDeployments[entities.EXT_DEPLOYMENT_METRICSERVER]; !isOK {
		return entities.ConditionInapplicable, "", nil, nil
	}
	wps := cc.GetWachPoints()
	if !checkMetricsServerHealthy(wps) {
		wrappedErr := fmt.Errorf("Cluster(%s)'s extensional deployment component: %s is not healthy yet!", cs.Id, entities.EXT_DEPLOYMENT_METRICSERVER)
		logrus.Error(wrappedErr)
		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	}
	err := cc.InitializeKubernetesClient()
	if err != nil {
		wrappedErr := errors.New("Failed to initialize Kubernetes client, error: " + err.Error())
		logrus.Error(wrappedErr)
		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	}
	nsi, err := cc.GetNodesInformation()
	if err != nil {
		wrappedErr := fmt.Errorf("Failed to list Kubernetes cluster %s nodes, error: %s", cc.GetSettings().Id, err.Error())
		logrus.Error(wrappedErr)
		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	}
	if len(nsi) == 0 {
		logrus.Warnf("Got empty value of Kubernetes cluster %s node list!", cc.GetSettings().Id)
		return entities.ConditionInapplicable, "", nil, nil
	}
	//synchronously call agent's API for injecting all of listed node information.
	err = generateSystemRoutingRules(agent, nsi)
	if err != nil {
		logrus.Errorf("Failed to make a call to the remote Lightning Monkey's agent instance: %s, error: %s", agent.Id, err.Error())
	}
	return entities.ConditionInapplicable, "", nil, nil
}

func generateSystemRoutingRules(agent entities.LightningMonkeyAgent, nodes []entities.KubernetesNodeInfo) error {
	gr := entities.GenerateSystemRoutingRulesRequest{Nodes: nodes}
	data, err := json.Marshal(gr)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/system/routes", agent.State.LastReportIP, agent.ListenPort), bytes.NewReader(data))
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

func checkMetricsServerHealthy(wps []entities.WatchPoint) bool {
	if wps == nil || len(wps) == 0 {
		return false
	}
	for _, v := range wps {
		//be careful that "v.Name" is the Kubernetes deployment name.
		if v.Name == "metrics-server" && v.Status == monitors.Healthy {
			return true
		}
	}
	return false
}
