package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"html/template"
	"strings"
)

const (
	//ensures version of kubeadm is "1.13.5"
	etcdConfigTemplate string = `apiVersion: "kubeadm.k8s.io/v1alpha3"
kind: ClusterConfiguration
etcd:
    local:
        image: {{.IMAGE}}
        dataDir: {{.DATADIR}}
        serverCertSANs:
        - "{{.HOST}}"
        - "127.0.0.1"
        peerCertSANs:
        - "{{.HOST}}"
        - "127.0.0.1"
        extraArgs:
            initial-cluster: {{.SERVERS}}
            initial-cluster-state: new
            name: {{.NAME}}
            listen-peer-urls: https://{{.ADDR}}:2380
            listen-client-urls: https://{{.ADDR}}:2379
            advertise-client-urls: https://{{.ADDR}}:2379
            initial-advertise-peer-urls: https://{{.ADDR}}:2380`
)

func HandleDeployETCD(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	if job.Arguments == nil || job.Arguments["addresses"] == "" {
		return false, xerrors.Errorf("Illegal ETCD deployment job, required arguments are missed %w", crashError)
	}
	servers := strings.Split(job.Arguments["addresses"], ",")
	var sb strings.Builder
	for i := 0; i < len(servers); i++ {
		ip := servers[i]
		if ip == *a.arg.Address {
			ip = "0.0.0.0"
		}
		name := generateETCDName(a, ip)
		sb.WriteString(fmt.Sprintf("%s=https://%s:2380", name, ip))
		if i != len(servers)-1 {
			sb.WriteString(",")
		}
	}
	serversConnection := sb.String()
	tmpl, err := template.New("etcd").Parse(etcdConfigTemplate)
	if err != nil {
		return false, xerrors.Errorf("Failed to parse ETCD configuration template, error: %s %w", err.Error(), crashError)
	}
	logrus.Infof("SERVER ADDR: %s", *a.arg.Address)
	args := map[string]string{
		"NAME":    generateETCDName(a, *a.arg.Address),
		"HOST":    *a.arg.Address,
		"SERVERS": serversConnection,
		"IMAGE":   a.basicImages.Images["etcd"].ImageName,
		"DATADIR": "/data/etcd",
		"ADDR":    "0.0.0.0",
	}
	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, args)
	if err != nil {
		return false, xerrors.Errorf("Failed to execute ETCD configuration template, error: %s %w", err.Error(), crashError)
	}
	err = certs.GenerateETCDClientCertificatesAndManifest(CERTIFICATE_STORAGE_PATH, buffer.String())
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
	for i := 0; i < len(containers); i++ {
		logrus.Infof("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if strings.Contains(containers[i].Names[0], "k8s_etcd") &&
			strings.Contains(containers[i].Names[0], "kube-system") &&
			strings.Contains(containers[i].Status, "Up") {
			return true, nil
		}
	}
	return false, nil
}

func generateETCDName(a *LightningMonkeyAgent, addr string) string {
	if addr == *a.arg.Address {
		addr = "0.0.0.0"
	}
	hasher := md5.New()
	hasher.Write([]byte([]byte(addr)))
	return hex.EncodeToString(hasher.Sum(nil))
}
