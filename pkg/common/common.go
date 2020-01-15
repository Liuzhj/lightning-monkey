package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/cache"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
)

var (
	StorageDriver  storage.LightningMonkeyStorageDriver
	ClusterManager cache.ClusterManagerInterface
	CertManager    certs.CertificateManager
	BasicImages    = map[string]entities.DockerImageCollection{
		"1.12.5": {
			DownloadType:      entities.DockerImageDownloadType_HTTP,
			HTTPDownloadToken: "",
			Images: map[string]entities.DockerImage{
				"etcd":       {ImageName: "docker.io/mirrorgooglecontainers/etcd:3.2.24", DownloadAddr: "%s/apis/v1/registry/1.12.5/etcd.tar?token=%s"},
				"k8s":        {ImageName: "g0194776/lightning-monkey-hyperkube:v1.12.5-2", DownloadAddr: "%s/apis/v1/registry/1.12.5/k8s.tar?token=%s"},
				"infra":      {ImageName: "docker.io/mirrorgooglecontainers/pause-amd64:3.1", DownloadAddr: "%s/apis/v1/registry/1.12.5/infra.tar?token=%s"},
				"coredns":    {ImageName: "docker.io/coredns/coredns:1.5.2", DownloadAddr: "%s/apis/v1/registry/software/coredns.tar?token=%s"},
				"ha":         {ImageName: "docker.io/pelin/haproxy-keepalived:latest", DownloadAddr: "%s/apis/v1/registry/software/ha.tar?token=%s"},
				"metrics":    {ImageName: "docker.io/mirrorgooglecontainers/metrics-server-amd64:v0.3.3", DownloadAddr: "%s/apis/v1/registry/software/metrics.tar?token=%s"},
				"traefik":    {ImageName: "docker.io/traefik:1.7.14", DownloadAddr: "%s/apis/v1/registry/software/traefik.tar?token=%s"},
				"router":     {ImageName: "docker.io/cloudnativelabs/kube-router:v0.2.5", DownloadAddr: "%s/apis/v1/registry/software/router.tar?token=%s"},
				"busybox":    {ImageName: "docker.io/busybox:latest", DownloadAddr: "%s/apis/v1/registry/software/busybox.tar?token=%s"},
				"prometheus": {ImageName: "docker.io/prom/prometheus:v2.2.1", DownloadAddr: "%s/apis/v1/registry/software/prometheus.tar?token=%s"},
				"es":         {ImageName: "docker.io/elasticsearch:6.8.3", DownloadAddr: "%s/apis/v1/registry/software/es.tar?token=%s"},
				"filebeat":   {ImageName: "docker.io/elastic/filebeat:6.8.3", DownloadAddr: "%s/apis/v1/registry/software/filebeat.tar?token=%s"},
				"helm":       {ImageName: "docker.io/fishead/gcr.io.kubernetes-helm.tiller:v2.12.3", DownloadAddr: "%s/apis/v1/registry/software/helmv2.tar?token=%s"},
			},
		},
		"1.13.12": {
			DownloadType:      entities.DockerImageDownloadType_HTTP,
			HTTPDownloadToken: "",
			Images: map[string]entities.DockerImage{
				"etcd":       {ImageName: "docker.io/mirrorgooglecontainers/etcd:3.2.24", DownloadAddr: "%s/apis/v1/registry/1.13.12/etcd.tar?token=%s"},
				"k8s":        {ImageName: "g0194776/lightning-monkey-hyperkube:v1.13.12", DownloadAddr: "%s/apis/v1/registry/1.13.12/k8s.tar?token=%s"},
				"infra":      {ImageName: "docker.io/mirrorgooglecontainers/pause-amd64:3.1", DownloadAddr: "%s/apis/v1/registry/1.13.12/infra.tar?token=%s"},
				"coredns":    {ImageName: "docker.io/coredns/coredns:1.5.2", DownloadAddr: "%s/apis/v1/registry/software/coredns.tar?token=%s"},
				"ha":         {ImageName: "docker.io/pelin/haproxy-keepalived:latest", DownloadAddr: "%s/apis/v1/registry/software/ha.tar?token=%s"},
				"metrics":    {ImageName: "docker.io/mirrorgooglecontainers/metrics-server-amd64:v0.3.3", DownloadAddr: "%s/apis/v1/registry/software/metrics.tar?token=%s"},
				"traefik":    {ImageName: "docker.io/traefik:1.7.14", DownloadAddr: "%s/apis/v1/registry/software/traefik.tar?token=%s"},
				"router":     {ImageName: "docker.io/cloudnativelabs/kube-router:v0.2.5", DownloadAddr: "%s/apis/v1/registry/software/router.tar?token=%s"},
				"busybox":    {ImageName: "docker.io/busybox:latest", DownloadAddr: "%s/apis/v1/registry/software/busybox.tar?token=%s"},
				"prometheus": {ImageName: "docker.io/prom/prometheus:v2.2.1", DownloadAddr: "%s/apis/v1/registry/software/prometheus.tar?token=%s"},
				"es":         {ImageName: "docker.io/elasticsearch:6.8.3", DownloadAddr: "%s/apis/v1/registry/software/es.tar?token=%s"},
				"filebeat":   {ImageName: "docker.io/elastic/filebeat:6.8.3", DownloadAddr: "%s/apis/v1/registry/software/filebeat.tar?token=%s"},
				"helm":       {ImageName: "docker.io/fishead/gcr.io.kubernetes-helm.tiller:v2.12.3", DownloadAddr: "%s/apis/v1/registry/software/helmv2.tar?token=%s"},
			},
		},
	}
)

func SaveAgent(agent *entities.LightningMonkeyAgent) (int64, error) {
	//STEP 1, save agent's settings.
	err := SaveAgentSettingsOnly(agent)
	if err != nil {
		return -1, err
	}
	//STEP 2, save agent's state with TTL.
	return SaveAgentStateOnlyWithTTL(agent.ClusterId, agent.Id, agent.State)
}

func SaveAgentSettingsOnly(agent *entities.LightningMonkeyAgent) error {
	ctx, cancel := context.WithTimeout(context.Background(), StorageDriver.GetRequestTimeoutDuration())
	defer cancel()
	//STEP 1, save agent's settings.
	path := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/settings", agent.ClusterId, agent.Id)
	data, err := json.Marshal(agent)
	if err != nil {
		return err
	}
	_, err = StorageDriver.Put(ctx, path, string(data))
	if err != nil {
		return err
	}
	return nil
}

func SaveAgentStateOnlyWithTTL(clusterId string, agentId string, state *entities.AgentState) (int64, error) {
	leaseId, err := newETCDLease()
	if err != nil {
		return -1, err
	}
	path := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/state", clusterId, agentId)
	data, err := json.Marshal(state)
	if err != nil {
		return -1, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), StorageDriver.GetRequestTimeoutDuration())
	defer cancel()
	_, err = StorageDriver.Put(ctx, path, string(data), clientv3.WithLease(clientv3.LeaseID(leaseId)))
	if err != nil {
		return -1, err
	}
	return leaseId, nil
}

func SaveAgentStateOnly(clusterId string, agentId string, leaseId int64, state *entities.AgentState) (int64, error) {
	var err error
	needRegenerateLease := leaseId == -1
	//STEP 1, renew/regenerate ETCD key lease.
	if needRegenerateLease {
		leaseId, err = newETCDLease()
		if err != nil {
			return -1, err
		}
		logrus.Infof("Agent %s has triggered reconnection procedure, state lease will renew one.", agentId)
	} else {
		lease := StorageDriver.NewLease()
		_, err := lease.KeepAliveOnce(context.TODO(), clientv3.LeaseID(leaseId))
		if err != nil {
			return -1, fmt.Errorf("Failed to renew lease to remote storage driver, error: %s", err.Error())
		}
	}
	//STEP 2, update state.
	path := fmt.Sprintf("/lightning-monkey/clusters/%s/agents/%s/state", clusterId, agentId)
	data, err := json.Marshal(state)
	if err != nil {
		return leaseId, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), StorageDriver.GetRequestTimeoutDuration())
	defer cancel()
	_, err = StorageDriver.Put(ctx, path, string(data), clientv3.WithLease(clientv3.LeaseID(leaseId)))
	return leaseId, err
}

func newETCDLease() (int64, error) {
	lease := StorageDriver.NewLease()
	grantRsp, err := lease.Grant(context.TODO(), 15)
	if err != nil {
		return -1, fmt.Errorf("Could not grant a new lease to remote storage driver, error: %s", err.Error())
	}
	return int64(grantRsp.ID), nil
}
