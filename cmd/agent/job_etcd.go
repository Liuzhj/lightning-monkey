package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"golang.org/x/xerrors"
	"html/template"
	"strings"
)

const (
	etcdConfigTemplate string = `apiVersion: "kubeadm.k8s.io/v1beta1"
kind: ClusterConfiguration
etcd:
    local:
        serverCertSANs:
        - "{{.HOST}}"
        peerCertSANs:
        - "{{.HOST}}"
        extraArgs:
            initial-cluster: {{.SERVERS}}
            initial-cluster-state: new
            name: {{.NAME}}
            listen-peer-urls: https://{{.HOST}}:2380
            listen-client-urls: https://{{.HOST}}:2379
            advertise-client-urls: https://{{.HOST}}:2379
            initial-advertise-peer-urls: https://{{.HOST}}:2380`
)

func HandleDeployETCD(job *entities.AgentJob, arg *AgentArgs) error {
	if job.Arguments == nil || job.Arguments["addresses"] == "" {
		return xerrors.Errorf("Illegal ETCD deployment job, required arguments are missed %w", crashError)
	}
	servers := strings.Split(job.Arguments["addresses"], ",")
	var sb strings.Builder
	for i := 0; i < len(servers); i++ {
		name := generateETCDName(servers[i])
		sb.WriteString(fmt.Sprintf("%s=https://%s:2380", name, servers[i]))
		if i != len(servers)-1 {
			sb.WriteString(",")
		}
	}
	serversConnection := sb.String()
	tmpl, err := template.New("etcd").Parse(etcdConfigTemplate)
	if err != nil {
		return xerrors.Errorf("Failed to parse ETCD configuration template, error: %s %w", err.Error(), crashError)
	}
	args := map[string]string{
		"NAME":    generateETCDName(*arg.Address),
		"HOST":    *arg.Address,
		"SERVERS": serversConnection,
	}

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, args)
	if err != nil {
		return xerrors.Errorf("Failed to execute ETCD configuration template, error: %s %w", err.Error(), crashError)
	}
	err = certs.GenerateETCDClientCertificatesAndManifest(CERTIFICATE_STORAGE_PATH, buffer.String())
	if err != nil {
		return xerrors.Errorf("Failed to generate ETCD client certificates, error: %s %w", err.Error(), crashError)
	}
	return nil
}

func generateETCDName(addr string) string {
	hasher := md5.New()
	hasher.Write([]byte([]byte(addr)))
	return hex.EncodeToString(hasher.Sum(nil))
}
