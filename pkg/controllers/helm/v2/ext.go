package v2

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/g0194776/lightningmonkey/pkg/utils"
	"github.com/sirupsen/logrus"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

const (
	payload = `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: helm
    name: tiller
  name: tiller-deploy
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: helm
      name: tiller
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: helm
        name: tiller
    spec:
      automountServiceAccountToken: true
      containers:
      - env:
        - name: TILLER_NAMESPACE
          value: kube-system
        - name: TILLER_HISTORY_MAX
          value: "0"
        image: fishead/gcr.io.kubernetes-helm.tiller:v2.12.3
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /liveness
            port: 44135
            scheme: HTTP
          initialDelaySeconds: 1
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
        name: tiller
        ports:
        - containerPort: 44134
          name: tiller
          protocol: TCP
        - containerPort: 44135
          name: http
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /readiness
            port: 44135
            scheme: HTTP
          initialDelaySeconds: 1
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 1
      dnsPolicy: ClusterFirst
      restartPolicy: IfNotPresent
      serviceAccount: tiller
      serviceAccountName: tiller
---
apiVersion: v1
kind: ServiceAccount
metadata:
 name: tiller
 namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
 name: tiller
roleRef:
 apiGroup: rbac.authorization.k8s.io
 kind: ClusterRole
 name: cluster-admin
subjects:
 - kind: ServiceAccount
   name: tiller
   namespace: kube-system
---
apiVersion: v1
kind: Service
metadata:
  labels:
    app: helm
    name: tiller
  name: tiller-deploy
  namespace: kube-system
spec:
  ports:
  - name: tiller
    port: 44134
    protocol: TCP
    targetPort: tiller
  selector:
    app: helm
    name: tiller
  sessionAffinity: None
  type: ClusterIP`
)

type HelmDeploymentController struct {
	client        *k8s.KubernetesClientSet
	settings      entities.LightningMonkeyClusterSettings
	parsedObjects []runtime.Object
}

func (dc *HelmDeploymentController) Initialize(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	dc.client = client
	dc.settings = settings
	yamlContentArr := strings.Split(payload, "---")
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

func (dc *HelmDeploymentController) Install() error {
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

func (dc *HelmDeploymentController) UnInstall() error {
	panic("implement me")
}

func (dc *HelmDeploymentController) GetName() string {
	return "Helm v2"
}

func (dc *HelmDeploymentController) HasInstalled() (bool, error) {
	if dc.settings.ExtensionalDeployments == nil || len(dc.settings.ExtensionalDeployments) == 0 {
		//skipping installation procedure.
		return true, nil
	}
	if _, isOK := dc.settings.ExtensionalDeployments[entities.EXT_DEPLOYMENT_HELM]; !isOK {
		//skipping installation procedure.
		return true, nil
	}
	ds, err := dc.client.CoreClient.ExtensionsV1beta1().Deployments("kube-system").Get("tiller-deploy", v1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed to retrieve Deployments(%s/%s) object from given Kubernetes cluster, error: %s", "kube-system", "tiller-deploy", err.Error())
	}
	return ds != nil, nil
}
