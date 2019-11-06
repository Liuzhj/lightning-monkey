package elasticsearch

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"
	"text/template"

	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/g0194776/lightningmonkey/pkg/utils"
	"github.com/sirupsen/logrus"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	Payload = `apiVersion: v1
kind: ServiceAccount
metadata:
 labels:
   app: elasticsearch
 name: elasticsearch-admin
 namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
 name: elasticsearch-admin
 labels:
   app: elasticsearch
roleRef:
 apiGroup: rbac.authorization.k8s.io
 kind: ClusterRole
 name: cluster-admin
subjects:
 - kind: ServiceAccount
   name: elasticsearch-admin
   namespace: kube-system
---
kind: Deployment
apiVersion: apps/v1
metadata:
 labels:
   app: elasticsearch
   role: master
 name: elasticsearch-master
 namespace: kube-system
spec:
 replicas: 3
 revisionHistoryLimit: 10
 selector:
   matchLabels:
     app: elasticsearch
     role: master
 template:
   metadata:
     labels:
       app: elasticsearch
       role: master
   spec:
     serviceAccountName: elasticsearch-admin
     containers:
       - name: elasticsearch-master
         image: elasticsearch:6.8.3
         command: ["bash", "-c", "ulimit -l unlimited && sysctl -w vm.max_map_count=262144 && exec su elasticsearch docker-entrypoint.sh"]
         ports:
           - containerPort: 9200
             protocol: TCP
           - containerPort: 9300
             protocol: TCP
         env:
           - name: "cluster.name"
             value: "elasticsearch-cluster"
           - name: "bootstrap.memory_lock"
             value: "true"
           - name: "discovery.zen.ping.unicast.hosts"
             value: "elasticsearch-discovery"
           - name: "discovery.zen.minimum_master_nodes"
             value: "2"
           - name: "discovery.zen.ping_timeout"
             value: "{{.PING_TIMEOUT}}"
           - name: "node.master"
             value: "true"
           - name: "node.data"
             value: "false"
           - name: "node.ingest"
             value: "false"
           - name: "ES_JAVA_OPTS"
             value: "-Xms{{.MASTER_XMS}} -Xmx{{.MASTER_XMX}}"
         securityContext:
           privileged: true
---
kind: Service
apiVersion: v1
metadata:
 labels:
   app: elasticsearch
 name: elasticsearch-discovery
 namespace: kube-system
spec:
 ports:
   - port: 9300
     targetPort: 9300
 selector:
   app: elasticsearch
   role: master
---
apiVersion: v1
kind: Service
metadata:
 name: elasticsearch-data-service
 namespace: kube-system
 labels:
   app: elasticsearch
   role: data
spec:
 ports:
   - port: 9200
     name: outer
   - port: 9300
     name: inner
 clusterIP: None
 selector:
   app: elasticsearch
   role: data
---
kind: StatefulSet
apiVersion: apps/v1
metadata:
 labels:
   app: elasticsearch
   role: data
 name: elasticsearch-data
 namespace: kube-system
spec:
 replicas: 5
 revisionHistoryLimit: 10
 selector:
   matchLabels:
     app: elasticsearch
 serviceName: elasticsearch-data-service
 template:
   metadata:
     labels:
       app: elasticsearch
       role: data
   spec:
     serviceAccountName: elasticsearch-admin
     affinity:
       podAntiAffinity:
         preferredDuringSchedulingIgnoredDuringExecution:
         - weight: 100
           podAffinityTerm:
             labelSelector:
               matchExpressions:
               - key: app
                 operator: In
                 values:
                 - elasticsearch
               - key: role
                 operator: In
                 values:
                 - data
             topologyKey: kubernetes.io/hostname
     containers:
       - name: elasticsearch-data
         image: elasticsearch:6.8.3
         command: ["bash", "-c", "ulimit -l unlimited && sysctl -w vm.max_map_count={{.MAX_MAP_COUNT}} && chown -R elasticsearch:elasticsearch /usr/share/elasticsearch/data && exec su elasticsearch docker-entrypoint.sh"]
         ports:
           - containerPort: 9200
             protocol: TCP
           - containerPort: 9300
             protocol: TCP
         env:
           - name: "cluster.name"
             value: "elasticsearch-cluster"
           - name: "bootstrap.memory_lock"
             value: "true"
           - name: "discovery.zen.ping.unicast.hosts"
             value: "elasticsearch-discovery"
           - name: "node.master"
             value: "false"
           - name: "node.data"
             value: "true"
           - name: "ES_JAVA_OPTS"
             value: "-Xms{{.DATA_XMS}} -Xmx{{.DATA_XMX}}"
         volumeMounts:
           - name: elasticsearch-data-volume
             mountPath: /usr/share/elasticsearch/data
         securityContext:
           privileged: true
     securityContext:
       fsGroup: 1000
     volumes:
       - name: elasticsearch-data-volume
         hostPath:
          path: {{.DATA_DIR}}
          type: DirectoryOrCreate
---
kind: Service
apiVersion: v1
metadata:
 labels:
   app: elasticsearch
 name: elasticsearch-service
 namespace: kube-system
spec:
 ports:
   - port: 9200
     targetPort: 9200
 selector:
   app: elasticsearch
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
 labels:
   app: elasticsearch
 name: elasticsearch-ingress
 namespace: kube-system
spec:
 rules:
   - host: elasticsearch.kube.com
     http:
       paths:
         - backend:
             serviceName: elasticsearch-service
             servicePort: 9200`
)

type ElasticSearchDeploymentController struct {
	client        *k8s.KubernetesClientSet
	settings      entities.LightningMonkeyClusterSettings
	parsedObjects []runtime.Object
	hasInstalled  int32
}

func (dc *ElasticSearchDeploymentController) Initialize(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	dc.client = client
	dc.settings = settings
	var isOK bool
	var args map[string]string
	if args, isOK = dc.settings.ExtensionalDeployments[entities.EXT_DEPLOYMENT_ES]; !isOK {
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
			return fmt.Errorf("Occurs unexpected exception during decoding yaml-based string from ElasticSearch deployment controller, error: %s", err.Error())
		}
		dc.parsedObjects = append(dc.parsedObjects, obj)
	}
	return nil
}

func (dc *ElasticSearchDeploymentController) processArguments(args map[string]string) string {
	var isOK bool
	var dataXMS, dataXMX, masterXMS, masterXMX, pingTimeout, dataDir, maxMapCount string
	if masterXMS, isOK = args[VAR_MASTER_XMS]; !isOK {
		masterXMS = "512m"
	}
	if masterXMX, isOK = args[VAR_MASTER_XMX]; !isOK {
		masterXMX = "512m"
	}
	if dataXMS, isOK = args[VAR_DATA_XMS]; !isOK {
		dataXMS = "1024m"
	}
	if dataXMX, isOK = args[VAR_DATA_XMX]; !isOK {
		dataXMX = "1024m"
	}
	if pingTimeout, isOK = args[VAR_PING_TIMEOUT]; !isOK {
		pingTimeout = "5s"
	}
	if dataDir, isOK = args[VAR_DATA_DIR]; !isOK {
		dataDir = "/data/elasticsearch"
	}
	if maxMapCount, isOK = args[VAR_MAX_MAP_COUNT]; !isOK {
		maxMapCount = "262144"
	}
	attributes := map[string]string{
		"DATA_XMS":      dataXMS,
		"DATA_XMX":      dataXMX,
		"MASTER_XMS":    masterXMS,
		"MASTER_XMX":    masterXMX,
		"PING_TIMEOUT":  pingTimeout,
		"DATA_DIR":      dataDir,
		"MAX_MAP_COUNT": maxMapCount,
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

func (dc *ElasticSearchDeploymentController) Install() error {
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

func (dc *ElasticSearchDeploymentController) UnInstall() error {
	panic("implement me")
}

func (dc *ElasticSearchDeploymentController) GetName() string {
	return "ElasticSearch"
}

func (dc *ElasticSearchDeploymentController) HasInstalled() (bool, error) {
	if dc.settings.ExtensionalDeployments == nil || len(dc.settings.ExtensionalDeployments) == 0 {
		//skipping installation procedure.
		return true, nil
	}
	if _, isOK := dc.settings.ExtensionalDeployments[entities.EXT_DEPLOYMENT_ES]; !isOK {
		//skipping installation procedure.
		return true, nil
	}
	if atomic.LoadInt32(&dc.hasInstalled) == 1 {
		return true, nil
	}
	ds, err := dc.client.CoreClient.AppsV1beta1().Deployments("kube-system").Get("elasticsearch-master", v1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed to retrieve Deployments(%s/%s) object from given Kubernetes cluster, error: %s", "kube-system", "elasticsearch-master", err.Error())
	}
	if ds != nil {
		atomic.StoreInt32(&dc.hasInstalled, 1)
	}
	return ds != nil, nil
}
