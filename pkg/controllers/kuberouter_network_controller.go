package controllers

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
	"k8s.io/client-go/kubernetes"
	"strings"
	"text/template"
)

const (
	kuberouter_deployment_payload = `apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-router-cfg
  namespace: kube-system
  labels:
    tier: node
    k8s-app: kube-router
data:
  cni-conf.json: |
    {
      "name":"kubernetes",
      "type":"bridge",
      "bridge":"kube-bridge",
      "isDefaultGateway":true,
      "hairpinMode":true,
      "ipam": {
        "type":"host-local"
      }
    }
  kubeconfig: |
    apiVersion: v1
    kind: Config
    clusterCIDR: "{{ .CIDR }}"
    clusters:
    - name: cluster
      cluster:
        certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        server: {{ .APISERVER }}
    users:
    - name: kube-router
      user:
        tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    contexts:
    - context:
        cluster: cluster
        user: kube-router
      name: kube-router-context
    current-context: kube-router-context
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  labels:
    k8s-app: kube-router
    tier: node
  name: kube-router
  namespace: kube-system
spec:
  template:
    metadata:
      labels:
        k8s-app: kube-router
        tier: node
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      serviceAccountName: kube-router
      containers:
      - name: kube-router
        image: cloudnativelabs/kube-router:v0.2.5
        imagePullPolicy: Always
        args:
        - "--run-router=true"
        - "--run-firewall=true"
        - "--run-service-proxy=true"
        - "--kubeconfig=/var/lib/kube-router/kubeconfig"
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        livenessProbe:
          httpGet:
            path: /healthz
            port: 20244
          initialDelaySeconds: 10
          periodSeconds: 3
        resources:
          requests:
            cpu: 250m
            memory: 250Mi
        securityContext:
          privileged: true
        volumeMounts:
        - name: lib-modules
          mountPath: /lib/modules
          readOnly: true
        - name: cni-conf-dir
          mountPath: /etc/cni/net.d
        - name: kubeconfig
          mountPath: /var/lib/kube-router
          readOnly: true
      initContainers:
      - name: install-cni
        image: busybox
        imagePullPolicy: Always
        command:
        - /bin/sh
        - -c
        - set -e -x;
          if [ ! -f /etc/cni/net.d/10-kuberouter.conf ]; then
            TMP=/etc/cni/net.d/.tmp-kuberouter-cfg;
            cp /etc/kube-router/cni-conf.json ${TMP};
            mv ${TMP} /etc/cni/net.d/10-kuberouter.conf;
          fi;
          if [ ! -f /var/lib/kube-router/kubeconfig ]; then
            TMP=/var/lib/kube-router/.tmp-kubeconfig;
            cp /etc/kube-router/kubeconfig ${TMP};
            mv ${TMP} /var/lib/kube-router/kubeconfig;
          fi
        volumeMounts:
        - mountPath: /etc/cni/net.d
          name: cni-conf-dir
        - mountPath: /etc/kube-router
          name: kube-router-cfg
        - name: kubeconfig
          mountPath: /var/lib/kube-router
      hostNetwork: true
      tolerations:
      - key: CriticalAddonsOnly
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      - effect: NoSchedule
        key: node.kubernetes.io/not-ready
        operator: Exists
      volumes:
      - name: lib-modules
        hostPath:
          path: /lib/modules
      - name: cni-conf-dir
        hostPath:
          path: /etc/cni/net.d
      - name: kube-router-cfg
        configMap:
          name: kube-router-cfg
      - name: kubeconfig
        hostPath:
          path: /var/lib/kube-router

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-router
  namespace: kube-system

---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: kube-router
  namespace: kube-system
rules:
  - apiGroups:
    - ""
    resources:
      - namespaces
      - pods
      - services
      - nodes
      - endpoints
    verbs:
      - list
      - get
      - watch
  - apiGroups:
    - "networking.k8s.io"
    resources:
      - networkpolicies
    verbs:
      - list
      - get
      - watch
  - apiGroups:
    - extensions
    resources:
      - networkpolicies
    verbs:
      - get
      - list
      - watch
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: kube-router
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kube-router
subjects:
- kind: ServiceAccount
  name: kube-router
  namespace: kube-system
`
)

type KubeRouterNetworkController struct {
	client        *kubernetes.Clientset
	settings      entities.LightningMonkeyClusterSettings
	parsedObjects []runtime.Object
}

func (nc *KubeRouterNetworkController) Initialize(client *kubernetes.Clientset, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	nc.client = client
	nc.settings = settings
	//TODO: use VIP.
	attributes := map[string]string{"CIDR": nc.settings.PodNetworkCIDR, "APISERVER": fmt.Sprintf("https://%s:6443", clientIp)}
	t := template.New("t1")
	t, err := t.Parse(kuberouter_deployment_payload)
	if err != nil {
		return fmt.Errorf("Failed to parse Kube-Router deployment metadata as golang template content, error: %s", err.Error())
	}
	buf := bytes.Buffer{}
	err = t.Execute(&buf, attributes)
	if err != nil {
		return fmt.Errorf("Failed to execute replacing procedure of golang template for Kube-Router deployment metadata, error: %s", err.Error())
	}
	yamlContentArr := strings.Split(buf.String(), "---")
	if yamlContentArr == nil || len(yamlContentArr) == 0 {
		return nil
	}
	for i := 0; i < len(yamlContentArr); i++ {
		obj, err := utils.DecodeYamlOrJson(yamlContentArr[i])
		if err != nil {
			return fmt.Errorf("Occurs unexpected exception during decoding yaml-based string from Kube-Router network stack controller, error: %s", err.Error())
		}
		nc.parsedObjects = append(nc.parsedObjects, obj)
	}
	return nil
}

func (nc *KubeRouterNetworkController) Install() error {
	if nc.parsedObjects == nil || len(nc.parsedObjects) == 0 {
		return nil
	}
	logrus.Infof("Start provisioning network stack for cluster: %s", nc.settings.Id)
	var err error
	var existed bool
	for i := 0; i < len(nc.parsedObjects); i++ {
		metadata, _ := utils.ObjectMetaFor(nc.parsedObjects[i])
		if existed, err = k8s.IsKubernetesResourceExists(nc.client, nc.parsedObjects[i]); err != nil && !k8sErr.IsNotFound(err) {
			return fmt.Errorf("Failed to check Kubernetes resource existence, error: %s", err.Error())
		} else if !existed {
			_, err = k8s.CreateK8SResource(nc.client, nc.parsedObjects[i])
			if err != nil {
				return fmt.Errorf("Failed to create Kubernetes resource: %s, error: %s", metadata.Name, err.Error())
			}
		}
		logrus.Infof("Kubernetes resource %s(%s) has been created successfully!", metadata.Name, nc.parsedObjects[i].GetObjectKind().GroupVersionKind().Kind)
	}
	return nil
}

func (nc *KubeRouterNetworkController) UnInstall() error {
	panic("implement me")
}

func (nc *KubeRouterNetworkController) GetName() string {
	return entities.AgentJob_Deploy_NetworkStack_KubeRouter
}

func (nc *KubeRouterNetworkController) HasInstalled() (bool, error) {
	ds, err := nc.client.ExtensionsV1beta1().DaemonSets("kube-system").Get("kube-router", v1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed to retrieve DaemonSet(%s/%s) object from given Kubernetes cluster, error: %s", "kube-system", "kube-router", err.Error())
	}
	return ds != nil, nil
}
