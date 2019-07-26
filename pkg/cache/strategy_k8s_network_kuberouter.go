package cache

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
)

type ClusterKubeRouterJobStrategy struct {
}

func (js *ClusterKubeRouterJobStrategy) GetStrategyName() string {
	return entities.AgentJob_Deploy_NetworkStack_KubeRouter
}

func (js *ClusterKubeRouterJobStrategy) CanDeploy(cc ClusterController, agent entities.LightningMonkeyAgent, cache *AgentCache) (entities.ConditionCheckedResult, string, map[string]string, error) {
	////STEP 1, lazy load, can directly skipped with internal initialization status.
	//err := cc.InitializeKubernetesClient()
	//var wrappedErr error
	//if err != nil {
	//	wrappedErr = errors.New("Failed to initialize Kubernetes client, error: " + err.Error())
	//	logrus.Error(wrappedErr)
	//	return entities.ConditionNotConfirmed, "", nil, wrappedErr
	//}
	////STEP 2, initialize cluster network controller, also can directly skipped with internal initialization status.
	//err = cc.InitializeNetworkController()
	//if err != nil {
	//	wrappedErr = fmt.Errorf("Failed to initialize cluster(%s) network controller, error: %s", cc.GetClusterId(), err.Error())
	//	logrus.Error(wrappedErr)
	//	return entities.ConditionNotConfirmed, "", nil, wrappedErr
	//}
	////Kubernetes client has been initialized successfully, try to commit YAML-formatted kube-router resource into cluster.
	//nc := cc.GetNetworkController()
	//if !nc.HasInstalled() {
	//	logrus.Infof("Try deploying network stack(%s) to cluster %s ......", nc.GetName(), cc.GetClusterId())
	//	err = cc.Install()
	//	if err != nil {
	//		wrappedErr = fmt.Errorf("Failed to deploy Kubernetes network stack(%s) to cluster %s, error: %s", nc.GetName(), cc.GetClusterId(), err.Error())
	//		logrus.Error(wrappedErr)
	//		return entities.ConditionNotConfirmed, "", nil, wrappedErr
	//	}
	//}
	return entities.ConditionInapplicable, "", nil, nil
}
