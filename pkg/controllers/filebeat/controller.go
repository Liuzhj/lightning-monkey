package filebeat

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
	"text/template"
)

const (
	Payload = `apiVersion: v1
kind: ConfigMap
metadata:
  name: filebeat-config
  namespace: kube-system
  labels:
    k8s-app: filebeat
data:
  filebeat.yml: |-
    filebeat.inputs:
    - type: log
      paths:
        - /var/lib/docker/containers/*/*.log
      processors:
        - add_kubernetes_metadata:
            host: ${NODE_NAME}
            matchers:
            - logs_path:
                logs_path: "/var/lib/docker/containers/"

    processors:
      - add_cloud_metadata:
      - add_host_metadata:

    cloud.id: ${ELASTIC_CLOUD_ID}
    cloud.auth: ${ELASTIC_CLOUD_AUTH}

    output.elasticsearch:
      hosts: ['${ELASTICSEARCH_HOST:elasticsearch}:${ELASTICSEARCH_PORT:9200}']
      # username: ${ELASTICSEARCH_USERNAME}
      # password: ${ELASTICSEARCH_PASSWORD}
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: filebeat
  namespace: kube-system
  labels:
    k8s-app: filebeat
spec:
  template:
    metadata:
      labels:
        k8s-app: filebeat
    spec:
      serviceAccountName: filebeat
      terminationGracePeriodSeconds: 30
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      containers:
      - name: filebeat
        image: elastic/filebeat:6.8.3
        args: [
          "-c", "/etc/filebeat.yml",
          "-e",
        ]
        env:
        - name: ELASTICSEARCH_HOST
          value: elasticsearch-service
        - name: ELASTICSEARCH_PORT
          value: "9200"
        - name: ELASTICSEARCH_USERNAME
          value:
        - name: ELASTICSEARCH_PASSWORD
          value:
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
          # If using Red Hat OpenShift uncomment this:
          #privileged: true
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 100Mi
        volumeMounts:
        - name: config
          mountPath: /etc/filebeat.yml
          readOnly: true
          subPath: filebeat.yml
        - name: data
          mountPath: /usr/share/filebeat/data
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
        - name: varlog
          mountPath: /var/log
          readOnly: true
      volumes:
      - name: config
        configMap:
          defaultMode: 0600
          name: filebeat-config
      - name: varlibdockercontainers
        hostPath:
          path: {{.CONTAINER_PATH}}
      - name: varlog
        hostPath:
          path: /var/log
      # data folder stores a registry of read status for all files, so we don't send everything again on a Filebeat pod restart
      - name: data
        hostPath:
          path: {{.STORAGE_PATH}}
          type: DirectoryOrCreate
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: filebeat
subjects:
- kind: ServiceAccount
  name: filebeat
  namespace: kube-system
roleRef:
  kind: ClusterRole
  name: filebeat
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: filebeat
  labels:
    k8s-app: filebeat
rules:
- apiGroups: [""] # "" indicates the core API group
  resources:
  - namespaces
  - pods
  verbs:
  - get
  - watch
  - list
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: filebeat
  namespace: kube-system
  labels:
    k8s-app: filebeat`
)

type FilebeatDeploymentController struct {
	client        *k8s.KubernetesClientSet
	settings      entities.LightningMonkeyClusterSettings
	parsedObjects []runtime.Object
}

func (dc *FilebeatDeploymentController) Initialize(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	dc.client = client
	dc.settings = settings
	var isOK bool
	var args map[string]string
	if args, isOK = dc.settings.ExtensionalDeployments[entities.EXT_DEPLOYMENT_FILEBEAT]; !isOK {
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

func (dc *FilebeatDeploymentController) processArguments(args map[string]string) string {
	var isOK bool
	var dockerContainerPath, filebeatStoragePath string
	if dockerContainerPath, isOK = args[VAR_DOCKER_CONTAINER_PATH]; !isOK {
		dockerContainerPath = "/var/lib/docker/containers"
	}
	if filebeatStoragePath, isOK = args[VAR_FILEBEAT_STORAGE_PATH]; !isOK {
		filebeatStoragePath = "/var/lib/filebeat-data"
	}
	attributes := map[string]string{
		"CONTAINER_PATH": dockerContainerPath,
		"STORAGE_PATH":   filebeatStoragePath,
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

func (dc *FilebeatDeploymentController) Install() error {
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

func (dc *FilebeatDeploymentController) UnInstall() error {
	panic("implement me")
}

func (dc *FilebeatDeploymentController) GetName() string {
	return "Filebeat"
}

func (dc *FilebeatDeploymentController) HasInstalled() (bool, error) {
	if dc.settings.ExtensionalDeployments == nil || len(dc.settings.ExtensionalDeployments) == 0 {
		//skipping installation procedure.
		return true, nil
	}
	if _, isOK := dc.settings.ExtensionalDeployments[entities.EXT_DEPLOYMENT_FILEBEAT]; !isOK {
		//skipping installation procedure.
		return true, nil
	}
	ds, err := dc.client.CoreClient.ExtensionsV1beta1().DaemonSets("kube-system").Get("filebeat", v1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed to retrieve DaemonSets(%s/%s) object from given Kubernetes cluster, error: %s", "kube-system", "filebeat", err.Error())
	}
	return ds != nil, nil
}
