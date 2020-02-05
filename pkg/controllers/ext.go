package controllers

import (
	"fmt"
	built_in "github.com/g0194776/lightningmonkey/pkg/controllers/built-in"
	"github.com/g0194776/lightningmonkey/pkg/controllers/elasticsearch"
	"github.com/g0194776/lightningmonkey/pkg/controllers/filebeat"
	v2 "github.com/g0194776/lightningmonkey/pkg/controllers/helm/v2"
	"github.com/g0194776/lightningmonkey/pkg/controllers/metricbeat"
	"github.com/g0194776/lightningmonkey/pkg/controllers/metrics"
	"github.com/g0194776/lightningmonkey/pkg/controllers/prometheus"
	"github.com/g0194776/lightningmonkey/pkg/controllers/traefik"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/sirupsen/logrus"
)

func CreateExtensionDeploymentController(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) (DeploymentController, error) {
	dc := ExtensionDeploymentController{client: client, settings: settings}
	err := dc.Initialize(client, clientIp, settings)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize extension deployment controller, error: %s", err.Error())
	}
	return &dc, nil
}

type ExtensionDeploymentController struct {
	client      *k8s.KubernetesClientSet
	settings    entities.LightningMonkeyClusterSettings
	controllers []DeploymentController
}

func (dc *ExtensionDeploymentController) Initialize(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	controllers := []DeploymentController{
		&built_in.DefaultImagePullingSecretsDeploymentController{},
		&prometheus.PrometheusDeploymentController{},
		&metrics.MetricServerDeploymentController{},
		&traefik.TraefikDeploymentController{},
		&elasticsearch.ElasticSearchDeploymentController{},
		&filebeat.FilebeatDeploymentController{},
		&metricbeat.MetricbeatDeploymentController{},
		&v2.HelmDeploymentController{},
	}
	dc.controllers = []DeploymentController{}
	var err error
	for i := 0; i < len(controllers); i++ {
		err = controllers[i].Initialize(client, clientIp, settings)
		if err != nil {
			logrus.Errorf("Failed to initialize %s deployment controller, error: %s", controllers[i].GetName(), err.Error())
			continue
		}
		dc.controllers = append(dc.controllers, controllers[i])
	}
	logrus.Infof("Registered extensional deployment controller count: %d", len(dc.controllers))
	return nil
}

func (dc *ExtensionDeploymentController) Install() error {
	if dc.controllers == nil || len(dc.controllers) == 0 {
		return nil
	}
	var err error
	for i := 0; i < len(dc.controllers); i++ {
		//pay attention that it'll loop all of registered controllers from the beginning, need to do more installation status check in the each of deployment controller.
		err = dc.controllers[i].Install()
		if err != nil {
			return fmt.Errorf("Failed to perform installation procedure to %s deployment controller, error: %s", dc.controllers[i].GetName(), err.Error())
		}
	}
	return nil
}

func (dc *ExtensionDeploymentController) UnInstall() error {
	panic("implement me")
}

func (dc *ExtensionDeploymentController) GetName() string {
	return "Extensibility"
}

func (dc *ExtensionDeploymentController) HasInstalled() (bool, error) {
	if dc.controllers == nil || len(dc.controllers) == 0 {
		return true, nil
	}
	var err error
	var hasInstalled bool
	for i := 0; i < len(dc.controllers); i++ {
		hasInstalled, err = dc.controllers[i].HasInstalled()
		if err != nil {
			return false, fmt.Errorf("Failed to check installation status from %s deployment controller, error: %s", dc.controllers[i].GetName(), err.Error())
		}
		if !hasInstalled {
			return hasInstalled, nil
		}
	}
	return true, nil
}
