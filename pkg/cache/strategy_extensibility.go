package cache

import (
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
)

type ExtensibilityDeploymentJobStrategy struct {
}

func (js *ExtensibilityDeploymentJobStrategy) GetStrategyName() string {
	return "Extensibility"
}

func (js *ExtensibilityDeploymentJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	//STEP 1, lazy load, can directly skipped with internal initialization status.
	err := cc.InitializeKubernetesClient()
	var wrappedErr error
	if err != nil {
		wrappedErr = errors.New("Failed to initialize Kubernetes client, error: " + err.Error())
		logrus.Error(wrappedErr)
		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	}
	//STEP 2, initialize extensibility deployment controller, also can directly skipped with internal initialization status.
	err = cc.InitializeExtensionDeploymentController()
	if err != nil {
		wrappedErr = fmt.Errorf("Failed to initialize cluster(%s) extensional deployment controller, error: %s", cc.GetClusterId(), err.Error())
		logrus.Error(wrappedErr)
		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	}
	nc := cc.GetExtensionDeploymentController()
	if hasInstalled, err := nc.HasInstalled(); err != nil {
		logrus.Error(err)
		return entities.ConditionNotConfirmed, "", nil, err
	} else if !hasInstalled {
		logrus.Infof("Try deploying extensional resource(%s) to cluster %s ......", nc.GetName(), cc.GetClusterId())
		err = nc.Install()
		if err != nil {
			wrappedErr = fmt.Errorf("Failed to deploy extensional resource(%s) to cluster %s, error: %s", nc.GetName(), cc.GetClusterId(), err.Error())
			logrus.Error(wrappedErr)
			return entities.ConditionNotConfirmed, "", nil, wrappedErr
		}
	}
	return entities.ConditionInapplicable, "", nil, nil
}
