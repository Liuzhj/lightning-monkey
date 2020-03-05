package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/docker/distribution/reference"
	"github.com/docker/engine-api/types"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/templates"
	"github.com/g0194776/lightningmonkey/pkg/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"html/template"
	"io"
	"strings"
)

func HandleDeployETCD(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	if job.Arguments == nil || job.Arguments["addresses"] == "" {
		return false, xerrors.Errorf("Illegal ETCD deployment job, required arguments are missed %w", crashError)
	}
	servers := strings.Split(job.Arguments["addresses"], ",")
	var sb strings.Builder
	for i := 0; i < len(servers); i++ {
		ip := servers[i]
		name := generateETCDName(a, ip)
		//if ip == *a.arg.Address {
		//	ip = "0.0.0.0"
		//}
		sb.WriteString(fmt.Sprintf("%s=https://%s:2380", name, ip))
		if i != len(servers)-1 {
			sb.WriteString(",")
		}
	}
	serversConnection := sb.String()
	logrus.Debugf("Try to get ETCD configuration template by Kubernetes version: %s", a.masterSettings[entities.MasterSettings_KubernetesVersion])
	ts, err := templates.GetTemplate(entities.AgentJob_Deploy_ETCD, a.masterSettings[entities.MasterSettings_KubernetesVersion])
	if err != nil {
		return false, xerrors.Errorf("Failed to get ETCD configuration template by given Kubernetes version \"%s\", error: %s %w", a.masterSettings[entities.MasterSettings_KubernetesVersion], err.Error(), crashError)
	}
	if ts == "" {
		return false, xerrors.Errorf("Got empty ETCD configuration template by given Kubernetes version \"%s\", %w", a.masterSettings[entities.MasterSettings_KubernetesVersion], crashError)
	}
	tmpl, err := template.New("etcd").Parse(ts)
	if err != nil {
		return false, xerrors.Errorf("Failed to parse ETCD configuration template, error: %s %w", err.Error(), crashError)
	}
	logrus.Infof("SERVER ADDR: %s", *a.arg.Address)
	ref, err := reference.Parse(a.basicImages.Images["etcd"].ImageName)
	if err != nil {
		return false, xerrors.Errorf("Failed to parse ETCD docker image reference object, reason: %s %w", err.Error(), crashError)
	}
	tagged := ref.(reference.NamedTagged)
	args := map[string]string{
		"NAME":      generateETCDName(a, *a.arg.Address),
		"HOST":      *a.arg.Address,
		"SERVERS":   serversConnection,
		"IMAGE":     a.basicImages.Images["etcd"].ImageName,
		"IMAGETAG":  tagged.Tag(),                                 //needed from Kubernetes v1.14
		"IMAGEREPO": utils.GetDockerRepositoryName(tagged.Name()), //needed from Kubernetes v1.14
		"DATADIR":   "/data/etcd",
		"ADDR":      *a.arg.Address, //"0.0.0.0",
	}
	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, args)
	if err != nil {
		return false, xerrors.Errorf("Failed to execute ETCD configuration template, error: %s %w", err.Error(), crashError)
	}
	err = common.CertManager.GenerateETCDClientCertificatesAndManifest(a.masterSettings[entities.MasterSettings_KubernetesVersion], CERTIFICATE_STORAGE_PATH, buffer.String())
	if err != nil {
		return false, xerrors.Errorf("Failed to generate ETCD client certificates, error: %s %w", err.Error(), crashError)
	}
	return true, nil
}

func CheckETCDHealth(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	var err error
	var containers []types.Container
	containers, err = a.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		logrus.Errorf("Failed to retrieve all containers information, error: %s", err.Error())
		return false, err
	}
	if containers == nil || len(containers) == 0 {
		return false, nil
	}
	var destContainerId string
	for i := 0; i < len(containers); i++ {
		logrus.Infof("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if strings.Contains(containers[i].Names[0], "k8s_etcd") &&
			strings.Contains(containers[i].Names[0], "kube-system") &&
			strings.Contains(containers[i].Status, "Up") {
			destContainerId = containers[i].ID
			break
		}
	}
	//unhealthy or not expected status.
	if destContainerId == "" {
		return false, nil
	}
	logrus.Debugf("Try performing ETCD health check with container-id: %s", destContainerId)
	result, err := getETCDClusterInfo(a, destContainerId)
	if err != nil {
		logrus.Errorf("Failed to perform ETCD health check, error: %s", err.Error())
		return false, nil
	}
	logrus.Debugf("ETCD health check result: \n%s", result)
	//return healthy status util expected count of ETCD nodes are ready.
	if strings.Count(string(result), "started") >= a.expectedETCDNodeCount {
		return true, nil
	}
	return false, nil
}

func generateETCDName(a *LightningMonkeyAgent, addr string) string {
	hasher := md5.New()
	hasher.Write([]byte([]byte(addr)))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getETCDClusterInfo(a *LightningMonkeyAgent, containerId string) (string, error) {
	//docker exec 01f sh -c  "export ETCDCTL_API=3 && /usr/local/bin/etcdctl --endpoints=https://[192.168.33.11]:2379 --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt --key=/etc/kubernetes/pki/etcd/healthcheck-client.key member list"
	cmdStr := fmt.Sprintf("export ETCDCTL_API=3 && /usr/local/bin/etcdctl --endpoints=https://[%s]:2379 --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/healthcheck-client.crt --key=/etc/kubernetes/pki/etcd/healthcheck-client.key member list", *a.arg.Address)
	config := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"/bin/sh", "-c", cmdStr},
	}
	execID, err := a.dockerClient.ContainerExecCreate(context.TODO(), containerId, config)
	if err != nil {
		return "", err
	}
	res, err := a.dockerClient.ContainerExecAttach(context.TODO(), execID.ID, types.ExecConfig{})
	if err != nil {
		return "", err
	}
	err = a.dockerClient.ContainerExecStart(context.TODO(), execID.ID, types.ExecStartCheck{})
	if err != nil {
		return "", err
	}
	sb := strings.Builder{}
	for {
		content, _, err := res.Reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			break
		}
		sb.WriteString(string(content) + "\n")
	}
	return sb.String(), nil
}
