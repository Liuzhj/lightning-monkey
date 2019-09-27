package cache

import (
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
)

type ClusterKubernetesDNSJobStrategy struct {
}

func (js *ClusterKubernetesDNSJobStrategy) GetStrategyName() string {
	return "DNS"
}

func (js *ClusterKubernetesDNSJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	//STEP 1, lazy load, can directly skipped with internal initialization status.
	err := cc.InitializeKubernetesClient()
	var wrappedErr error
	if err != nil {
		wrappedErr = errors.New("Failed to initialize Kubernetes client, error: " + err.Error())
		logrus.Error(wrappedErr)
		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	}
	//STEP 2, initialize cluster DNS deployment controller, also can directly skipped with internal initialization status.
	err = cc.InitializeDNSController()
	if err != nil {
		wrappedErr = fmt.Errorf("Failed to initialize cluster(%s) DNS deployment controller, error: %s", cc.GetClusterId(), err.Error())
		logrus.Error(wrappedErr)
		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	}
	//Kubernetes client has been initialized successfully, try to commit YAML-formatted kube-router resource into cluster.
	dc := cc.GetDNSController()
	if hasInstalled, err := dc.HasInstalled(); err != nil {
		logrus.Error(err)
		return entities.ConditionNotConfirmed, "", nil, err
	} else if !hasInstalled {
		logrus.Infof("Try deploying DNS(%s) to cluster %s ......", dc.GetName(), cc.GetClusterId())
		err = dc.Install()
		if err != nil {
			wrappedErr = fmt.Errorf("Failed to deploy Kubernetes DNS(%s) to cluster %s, error: %s", dc.GetName(), cc.GetClusterId(), err.Error())
			logrus.Error(wrappedErr)
			return entities.ConditionNotConfirmed, "", nil, wrappedErr
		}
	}
	return entities.ConditionInapplicable, "", nil, nil
}
