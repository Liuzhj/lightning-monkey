package controllers

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"
)

func CreateExtensionDeploymentController(client *k8s.Clientset, clientIp string, settings entities.LightningMonkeyClusterSettings) (DeploymentController, error) {
	dc := ExtensionDeploymentController{client: client, settings: settings}
	err := dc.Initialize(client, clientIp, settings)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize extension deployment controller, error: %s", err.Error())
	}
	return &dc, nil
}

type ExtensionDeploymentController struct {
	client      *k8s.Clientset
	settings    entities.LightningMonkeyClusterSettings
	controllers []DeploymentController
}

func (dc *ExtensionDeploymentController) Initialize(client *k8s.Clientset, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	controllers := []DeploymentController{
		&PrometheusDeploymentController{},
		&MetricServerDeploymentController{},
	}
	var err error
	for i := 0; i < len(controllers); i++ {
		err = controllers[i].Initialize(client, clientIp, settings)
		if err != nil {
			return fmt.Errorf("Failed to initialize %s deployment controller, error: %s", controllers[i].GetName(), err.Error())
		}
	}
	dc.controllers = controllers
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
