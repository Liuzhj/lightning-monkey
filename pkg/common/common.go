package common

import (
	"github.com/g0194776/lightningmonkey/pkg/cache"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/storage"
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
