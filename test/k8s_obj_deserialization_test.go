package test

import (
	"github.com/g0194776/lightningmonkey/pkg/utils"
	assert "github.com/stretchr/testify/require"
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
