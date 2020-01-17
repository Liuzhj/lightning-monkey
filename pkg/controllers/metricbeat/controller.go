package metricbeat

import (
	"bytes"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/g0194776/lightningmonkey/pkg/utils"
	"github.com/sirupsen/logrus"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
	"sync/atomic"
	"text/template"
)

const (
	Payload = `apiVersion: v1
kind: ConfigMap
metadata:
  name: metricbeat-deployment-config
  namespace: kube-system
  labels:
    k8s-app: metricbeat
data:
  metricbeat.yml: |-
    metricbeat.config.modules:
      path: ${path.config}/modules.d/*.yml
      reload.enabled: false

    # processors:
    #   - add_cloud_metadata:

    # cloud.id: ${ELASTIC_CLOUD_ID}
    # cloud.auth: ${ELASTIC_CLOUD_AUTH}

    output.elasticsearch:
      hosts: ['${ELASTICSEARCH_HOST:elasticsearch}:${ELASTICSEARCH_PORT:9200}']
      index: "k8s-prod-%{[beat.version]}-%{+yyyy.MM.dd}"
      # username: ${ELASTICSEARCH_USERNAME}
      # password: ${ELASTICSEARCH_PASSWORD}
    setup.template:
      name: 'k8s-prod'
      pattern: 'k8s-prod-*'
      enabled: false
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: metricbeat-deployment-modules
  namespace: kube-system
  labels:
    k8s-app: metricbeat
    kubernetes.io/cluster-service: "true"
data:
  kubernetes.yml: |-
    - module: kubernetes
      enabled: true
      metricsets:
        - event
---
# Deploy singleton instance in the whole cluster for some unique data sources, like kube-state-metrics
apiVersion: apps/v1
kind: Deployment
metadata:
  name: metricbeat
  namespace: kube-system
  labels:
    k8s-app: metricbeat
spec:
  selector:
    matchLabels:
      k8s-app: metricbeat
  template:
    metadata:
      labels:
        k8s-app: metricbeat
    spec:
      serviceAccountName: metricbeat
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      containers:
      - name: metricbeat
        image: docker.io/elastic/metricbeat:6.8.3
        args: [
          "-c", "/etc/metricbeat.yml",
          "-e",
        ]
        env:
        - name: ELASTICSEARCH_HOST
          value: {{.ES_HOST}}
        - name: ELASTICSEARCH_PORT
          value: {{.ES_PORT}}
        - name: ELASTICSEARCH_USERNAME
          value: {{.ES_USERNAME}}
        - name: ELASTICSEARCH_PASSWORD
          value: {{.ES_PASSWORD}}
        - name: ELASTIC_CLOUD_ID
          value:
        - name: ELASTIC_CLOUD_AUTH
          value:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        securityContext:
          runAsUser: 0
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 100Mi
        volumeMounts:
        - name: config
          mountPath: /etc/metricbeat.yml
          readOnly: true
          subPath: metricbeat.yml
        - name: modules
          mountPath: /usr/share/metricbeat/modules.d
          readOnly: true
      volumes:
      - name: config
        configMap:
          defaultMode: 0600
          name: metricbeat-deployment-config
      - name: modules
        configMap:
          defaultMode: 0600
          name: metricbeat-deployment-modules
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: metricbeat
subjects:
- kind: ServiceAccount
  name: metricbeat
  namespace: kube-system
roleRef:
  kind: ClusterRole
  name: metricbeat
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: metricbeat
  labels:
    k8s-app: metricbeat
rules:
- apiGroups: [""]
  resources:
  - nodes
  - namespaces
  - events
  - pods
  verbs: ["get", "list", "watch"]
- apiGroups: ["extensions"]
  resources:
  - replicasets
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources:
  - statefulsets
  - deployments
  verbs: ["get", "list", "watch"]
- apiGroups:
  - ""
  resources:
  - nodes/stats
  verbs:
  - get
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: metricbeat
  namespace: kube-system
  labels:
    k8s-app: metricbeat
`
)

type MetricbeatDeploymentController struct {
	client        *k8s.KubernetesClientSet
	settings      entities.LightningMonkeyClusterSettings
	parsedObjects []runtime.Object
	hasInstalled  int32
}

func (dc *MetricbeatDeploymentController) Initialize(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	dc.client = client
	dc.settings = settings
	var isOK bool
	var args map[string]string
	if args, isOK = dc.settings.ExtensionalDeployments[entities.EXT_DEPLOYMENT_METRICBEAT]; !isOK {
		//skipping installation procedure.
		return nil
	}
	//replace variables.
	yamlContentArr := strings.Split(dc.processArguments(args), "---")
	if yamlContentArr == nil || len(yamlContentArr) == 0 {
		return nil
	}
	for i := 0; i < len(yamlContentArr); i++ {
		obj, err := utils.DecodeYamlOrJson(yamlContentArr[i])
		if err != nil {
			return fmt.Errorf("Occurs unexpected exception during decoding yaml-based string from %s deployment controller, error: %s", dc.GetName(), err.Error())
		}
		dc.parsedObjects = append(dc.parsedObjects, obj)
	}
	return nil
}

func (dc *MetricbeatDeploymentController) processArguments(args map[string]string) string {
	var isOK bool
	var esHost, esPort, esUsername, esPassword string

	if esHost, isOK = args[VAR_ES_HOST]; !isOK {
		esHost = "elasticsearch-service"
	}
	if esPort, isOK = args[VAR_ES_PORT]; !isOK {
		esPort = "9200"
	}
	if esUsername, isOK = args[VAR_ES_USERNAME]; !isOK {
		esUsername = ""
	}
	if esPassword, isOK = args[VAR_ES_PASSWORD]; !isOK {
		esPassword = ""
	}

	attributes := map[string]string{
		"ES_HOST":     esHost,
		"ES_PORT":     esPort,
		"ES_USERNAME": esUsername,
		"ES_PASSWORD": esPassword,
	}
	t := template.New("t1")
	t, err := t.Parse(Payload)
	if err != nil {
		logrus.Errorf("Failed to parse %s deployment metadata as golang template content, error: %s", dc.GetName(), err.Error())
		return ""
	}
	buf := bytes.Buffer{}
	err = t.Execute(&buf, attributes)
	if err != nil {
		logrus.Errorf("Failed to execute replacing procedure of golang template for %s deployment metadata, error: %s", dc.GetName(), err.Error())
		return ""
	}
	return buf.String()
}

func (dc *MetricbeatDeploymentController) Install() error {
	if dc.parsedObjects == nil || len(dc.parsedObjects) == 0 {
		return nil
	}
	var err error
	var existed bool
	var hasInstalled bool
	hasInstalled, err = dc.HasInstalled()
	if err != nil {
		return fmt.Errorf("Failed to check installation status in the %s deployment controller, error: %s", dc.GetName(), err.Error())
	}
	//duplicated installation action, ignore.
	if hasInstalled {
		return nil
	}
	logrus.Infof("Start provisioning %s for cluster: %s", dc.GetName(), dc.settings.Id)
	for i := 0; i < len(dc.parsedObjects); i++ {
		metadata, err := utils.ObjectMetaFor(dc.parsedObjects[i])
		if err != nil {
			return fmt.Errorf("Failed to get Kubernetes resource, error: %s", err.Error())
		}
		if existed, err = k8s.IsKubernetesResourceExists(dc.client, dc.parsedObjects[i]); err != nil && !k8sErr.IsNotFound(err) {
			return fmt.Errorf("Failed to check Kubernetes resource existence, error: %s", err.Error())
		} else if !existed {
			_, err = k8s.CreateK8SResource(dc.client, dc.parsedObjects[i])
			if err != nil {
				return fmt.Errorf("Failed to create Kubernetes resource: %s, error: %s", metadata.Name, err.Error())
			}
		}
		logrus.Infof("Kubernetes resource %s(%s) has been created successfully!", metadata.Name, dc.parsedObjects[i].GetObjectKind().GroupVersionKind().Kind)
	}
	return nil
}

func (dc *MetricbeatDeploymentController) UnInstall() error {
	panic("implement me")
}

func (dc *MetricbeatDeploymentController) GetName() string {
	return "Metricbeat"
}

func (dc *MetricbeatDeploymentController) HasInstalled() (bool, error) {
	if dc.settings.ExtensionalDeployments == nil || len(dc.settings.ExtensionalDeployments) == 0 {
		//skipping installation procedure.
		return true, nil
	}
	if _, isOK := dc.settings.ExtensionalDeployments[entities.EXT_DEPLOYMENT_METRICBEAT]; !isOK {
		//skipping installation procedure.
		return true, nil
	}
	if atomic.LoadInt32(&dc.hasInstalled) == 1 {
		return true, nil
	}
	ds, err := dc.client.CoreClient.ExtensionsV1beta1().Deployments("kube-system").Get("metricbeat", v1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed to retrieve Deployments(%s/%s) object from given Kubernetes cluster, error: %s", "kube-system", "metricbeat", err.Error())
	}
	if ds != nil {
		atomic.StoreInt32(&dc.hasInstalled, 1)
	}
	return ds != nil, nil
}
