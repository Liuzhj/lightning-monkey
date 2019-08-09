package controllers

import (
	"bytes"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/g0194776/lightningmonkey/pkg/utils"
	"github.com/sirupsen/logrus"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"strings"
	"text/template"
)

const (
	coredns_deployment_payload = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: coredns
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
rules:
- apiGroups:
  - ""
  resources:
  - endpoints
  - services
  - pods
  - namespaces
  verbs:
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - nodes
  verbs:
  - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:coredns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:coredns
subjects:
- kind: ServiceAccount
  name: coredns
  namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health
        ready
        kubernetes {{.DOMAIN}} in-addr.arpa ip6.arpa {
          pods insecure
          fallthrough in-addr.arpa ip6.arpa
        }
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: kube-dns
  template:
    metadata:
      labels:
        k8s-app: kube-dns
    spec:
      priorityClassName: system-cluster-critical
      serviceAccountName: coredns
      tolerations:
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      nodeSelector:
        beta.kubernetes.io/os: linux
      containers:
      - name: coredns
        image: coredns/coredns:1.5.2
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            memory: 170Mi
          requests:
            cpu: 100m
            memory: 70Mi
        args: [ "-conf", "/etc/coredns/Corefile" ]
        volumeMounts:
        - name: config-volume
          mountPath: /etc/coredns
          readOnly: true
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
        - containerPort: 9153
          name: metrics
          protocol: TCP
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_BIND_SERVICE
            drop:
            - all
          readOnlyRootFilesystem: true
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
        readinessProbe:
          httpGet:
            path: /ready
            port: 8181
            scheme: HTTP
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
            - key: Corefile
              path: Corefile
---
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  selector:
    k8s-app: kube-dns
  clusterIP: {{.DNSIP}}
  ports:
  - name: dns
    port: 53
    protocol: UDP
  - name: dns-tcp
    port: 53
    protocol: TCP
  - name: metrics
    port: 9153
    protocol: TCP`
)

type CoreDNSController struct {
	client        *kubernetes.Clientset
	settings      entities.LightningMonkeyClusterSettings
	parsedObjects []runtime.Object
}

func (dc *CoreDNSController) Initialize(client *kubernetes.Clientset, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	dc.client = client
	dc.settings = settings
	attributes := map[string]string{
		"DOMAIN": dc.settings.ServiceDNSDomain,
		"DNSIP":  dc.settings.ServiceDNSClusterIP,
	}
	t := template.New("t1")
	t, err := t.Parse(coredns_deployment_payload)
	if err != nil {
		return fmt.Errorf("Failed to parse CoreDNS deployment metadata as golang template content, error: %s", err.Error())
	}
	buf := bytes.Buffer{}
	err = t.Execute(&buf, attributes)
	if err != nil {
		return fmt.Errorf("Failed to execute replacing procedure of golang template for CoreDNS deployment metadata, error: %s", err.Error())
	}
	yamlContentArr := strings.Split(buf.String(), "---")
	if yamlContentArr == nil || len(yamlContentArr) == 0 {
		return nil
	}
	for i := 0; i < len(yamlContentArr); i++ {
		obj, err := utils.DecodeYamlOrJson(yamlContentArr[i])
		if err != nil {
			return fmt.Errorf("Occurs unexpected exception during decoding yaml-based string from CoreDNS network stack controller, error: %s", err.Error())
		}
		dc.parsedObjects = append(dc.parsedObjects, obj)
	}
	return nil
}

func (dc *CoreDNSController) Install() error {
	if dc.parsedObjects == nil || len(dc.parsedObjects) == 0 {
		return nil
	}
	logrus.Infof("Start provisioning DNS(%s) for cluster: %s", dc.GetName(), dc.settings.Id)
	var err error
	var existed bool
	for i := 0; i < len(dc.parsedObjects); i++ {
		metadata, _ := meta_v1.ObjectMetaFor(dc.parsedObjects[i])
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

func (cd *CoreDNSController) UnInstall() error {
	panic("implement me")
}

func (dc *CoreDNSController) GetName() string {
	return "CoreDNS"
}

func (dc *CoreDNSController) HasInstalled() (bool, error) {
	ds, err := dc.client.AppsV1beta1().Deployments("kube-system").Get("coredns", v1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed to retrieve Deployments(%s/%s) object from given Kubernetes cluster, error: %s", "kube-system", "coredns", err.Error())
	}
	return ds != nil, nil
}
