package test

import (
	"github.com/g0194776/lightningmonkey/pkg/utils"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
	"testing"
)

func Test_DeserializeAPIServiceObject(t *testing.T) {
	service := `---
    apiVersion: apiregistration.k8s.io/v1beta1
    kind: APIService
    metadata:
      name: v1beta1.metrics.k8s.io
    spec:
     service:
       name: metrics-server
       namespace: kube-system
     group: metrics.k8s.io
     version: v1beta1
     insecureSkipTLSVerify: true
     groupPriorityMinimum: 100
     versionPriority: 100`
	obj, err := utils.DecodeYamlOrJson(service)
	if err != nil {
		panic(err)
	}
	assert.NotNil(t, obj)
}

func Test_DeserializeComplexYaml(t *testing.T) {
	payload := `kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:aggregated-metrics-reader
  labels:
    rbac.authorization.k8s.io/aggregate-to-view: "true"
    rbac.authorization.k8s.io/aggregate-to-edit: "true"
    rbac.authorization.k8s.io/aggregate-to-admin: "true"
rules:
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: metrics-server:system:auth-delegator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: metrics-server-auth-reader
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: extension-apiserver-authentication-reader
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system
---
apiVersion: apiregistration.k8s.io/v1beta1
kind: APIService
metadata:
  name: v1beta1.metrics.k8s.io
spec:
 service:
   name: metrics-server
   namespace: kube-system
 group: metrics.k8s.io
 version: v1beta1
 insecureSkipTLSVerify: true
 groupPriorityMinimum: 100
 versionPriority: 100
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: metrics-server
  namespace: kube-system
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: metrics-server
  namespace: kube-system
  labels:
    k8s-app: metrics-server
spec:
  selector:
    matchLabels:
      k8s-app: metrics-server
  template:
    metadata:
      name: metrics-server
      labels:
        k8s-app: metrics-server
    spec:
      serviceAccountName: metrics-server
      volumes:
      # mount in tmp so we can safely use from-scratch images and/or read-only containers
      - name: tmp-dir
        emptyDir: {}
      containers:
      - name: metrics-server
        image: mirrorgooglecontainers/metrics-server-amd64:v0.3.3
        args:
        - --metric-resolution=1m
        - --kubelet-preferred-address-types=InternalIP,Hostname,InternalDNS,ExternalDNS,ExternalIP
        imagePullPolicy: Always
        volumeMounts:
        - name: tmp-dir
          mountPath: /tmp
---
apiVersion: v1
kind: Service
metadata:
  name: metrics-server
  namespace: kube-system
  labels:
    kubernetes.io/name: "Metrics-server"
    kubernetes.io/cluster-service: "true"
spec:
  selector:
    k8s-app: metrics-server
  ports:
  - port: 443
    protocol: TCP
    targetPort: 443
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:metrics-server
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - nodes
  - nodes/stats
  - namespaces
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:metrics-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:metrics-server
subjects:
- kind: ServiceAccount
  name: metrics-server
  namespace: kube-system`
	var parsedObjects []runtime.Object
	yamlContentArr := strings.Split(payload, "---")
	for i := 0; i < len(yamlContentArr); i++ {
		obj, err := utils.DecodeYamlOrJson(yamlContentArr[i])
		if err != nil {
			panic(err)
		}
		parsedObjects = append(parsedObjects, obj)
	}
	assert.True(t, len(parsedObjects) == len(yamlContentArr))
}

//func Test_GetAPIServie(t *testing.T) {
//	service := `---
//    apiVersion: apiregistration.k8s.io/v1beta1
//    kind: APIService
//    metadata:
//      name: v1beta1.metrics.k8s.io
//    spec:
//     service:
//       name: metrics-server
//       namespace: kube-system
//     group: metrics.k8s.io
//     version: v1beta1
//     insecureSkipTLSVerify: true
//     groupPriorityMinimum: 100
//     versionPriority: 100`
//	obj, err := utils.DecodeYamlOrJson(service)
//	if err != nil {
//		panic(err)
//	}
//	assert.NotNil(t, obj)
//	client, err := k8s.NewForConfig(&rest.Config{
//		Host: "http://10.203.40.21:8080",
//	})
//	if err != nil {
//		panic(err)
//	}
//	client2, err := agg_v1beta.NewForConfig(&rest.Config{
//		Host: "http://10.203.40.21:8080",
//	})
//	if err != nil {
//		panic(err)
//	}
//	_, err = k8s_helper.IsKubernetesResourceExists(&k8s_helper.KubernetesClientSet{CoreClient: client, APIRegClientV1beta1: client2}, obj)
//	if err != nil {
//		if k8sErr.IsNotFound(err) {
//			return
//		}
//		panic(err)
//	}
//}
