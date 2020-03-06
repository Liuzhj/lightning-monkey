package controllers

import (
	built_in "github.com/g0194776/lightningmonkey/pkg/controllers/built-in"
	"github.com/g0194776/lightningmonkey/pkg/controllers/elasticsearch"
	"github.com/g0194776/lightningmonkey/pkg/controllers/filebeat"
	v2 "github.com/g0194776/lightningmonkey/pkg/controllers/helm/v2"
	"github.com/g0194776/lightningmonkey/pkg/controllers/metricbeat"
	"github.com/g0194776/lightningmonkey/pkg/controllers/metrics"
	"github.com/g0194776/lightningmonkey/pkg/controllers/prometheus"
	"github.com/g0194776/lightningmonkey/pkg/controllers/traefik"
	"github.com/g0194776/lightningmonkey/pkg/entities"
)

func init() {
	if ecc == nil {
		ecc = make(map[string]extControllerSetting)
	}
	registerExtDeploymentController("BUILT-IN-SECRETE", func() DeploymentController { return &built_in.DefaultImagePullingSecretsDeploymentController{} }, "*")
	registerExtDeploymentController(entities.EXT_DEPLOYMENT_PROMETHEUS, func() DeploymentController { return &prometheus.PrometheusDeploymentController{} }, "*")
	registerExtDeploymentController(entities.EXT_DEPLOYMENT_METRICSERVER, func() DeploymentController { return &metrics.MetricServerDeploymentController{} }, "*")
	registerExtDeploymentController(entities.EXT_DEPLOYMENT_TRAEFIK, func() DeploymentController { return &traefik.TraefikDeploymentController{} }, "*")
	registerExtDeploymentController(entities.EXT_DEPLOYMENT_ES, func() DeploymentController { return &elasticsearch.ElasticSearchDeploymentController{} }, "*")
	registerExtDeploymentController(entities.EXT_DEPLOYMENT_FILEBEAT, func() DeploymentController { return &filebeat.FilebeatDeploymentController{} }, "*")
	registerExtDeploymentController(entities.EXT_DEPLOYMENT_METRICBEAT, func() DeploymentController { return &metricbeat.MetricbeatDeploymentController{} }, "*")
	registerExtDeploymentController(entities.EXT_DEPLOYMENT_HELM, func() DeploymentController { return &v2.HelmDeploymentController{} }, "*")
}
