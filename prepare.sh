#!/bin/bash

SUPPORTED_K8S_VERSIONS=("1.12.10" "1.13.12" "1.14.10" "1.15.9")

echo "Downloading RPM packages..."
wget https://yum.dockerproject.org/repo/main/centos/7/Packages/docker-engine-1.12.6-1.el7.centos.x86_64.rpm
wget https://yum.dockerproject.org/repo/main/centos/7/Packages/docker-engine-selinux-1.12.6-1.el7.centos.noarch.rpm
wget http://mirror.rc.usf.edu/compute_lock/elrepo/kernel/el7/x86_64/RPMS/kernel-ml-4.15.6-1.el7.elrepo.x86_64.rpm
wget https://get.helm.sh/helm-v2.12.3-linux-amd64.tar.gz

./download_kubeadm.sh "${SUPPORTED_K8S_VERSIONS[*]}"

echo "Downloading depended Docker images..."
./download-frozen-image-v2.sh img_etcd mirrorgooglecontainers/etcd:3.2.24
./download-frozen-image-v2.sh img_infra mirrorgooglecontainers/pause-amd64:3.1
./download-frozen-image-v2.sh img_coredns coredns/coredns:1.5.2
./download-frozen-image-v2.sh img_ha pelin/haproxy-keepalived:latest
./download-frozen-image-v2.sh img_metrics mirrorgooglecontainers/metrics-server-amd64:v0.3.3
./download-frozen-image-v2.sh img_traefik traefik:1.7.14
./download-frozen-image-v2.sh img_router cloudnativelabs/kube-router:v0.2.5
./download-frozen-image-v2.sh img_busybox busybox:latest
./download-frozen-image-v2.sh img_prometheus prom/prometheus:v2.2.1
./download-frozen-image-v2.sh img_lmagent g0194776/lightning-monkey-agent:latest
./download-frozen-image-v2.sh img_exporter prom/node-exporter:v0.15.2
./download-frozen-image-v2.sh img_es elasticsearch:6.8.3
./download-frozen-image-v2.sh img_filebeat elastic/filebeat:6.8.3
./download-frozen-image-v2.sh img_helmv2 fishead/gcr.io.kubernetes-helm.tiller:v2.12.3
./download-frozen-image-v2.sh img_metricbeat elastic/metricbeat:6.8.3

echo "Downloading specified versions of kubeadm..."
for ver in ${SUPPORTED_K8S_VERSIONS[*]} ; do
    ./download-frozen-image-v2.sh img_hyperkube_${ver} g0194776/lightning-monkey-hyperkube:v${ver}
    tar -C 'img_hyperkube_${ver}' -cf ./k8s_${ver}.tar .
done

echo "Assmebling downloaded Docker image layers to tarball files..."
tar -C 'img_etcd' -cf ./etcd.tar .
tar -C 'img_infra' -cf ./infra.tar .
tar -C 'img_coredns' -cf ./coredns.tar .
tar -C 'img_ha' -cf ./ha.tar .
tar -C 'img_metrics' -cf ./metrics.tar .
tar -C 'img_traefik' -cf ./traefik.tar .
tar -C 'img_router' -cf ./router.tar .
tar -C 'img_busybox' -cf ./busybox.tar .
tar -C 'img_prometheus' -cf ./prometheus.tar .
tar -C 'img_lmagent' -cf ./lmagent.tar .
tar -C 'img_exporter' -cf ./exporter.tar .
tar -C 'img_es' -cf ./es.tar .
tar -C 'img_filebeat' -cf ./filebeat.tar .
tar -C 'img_metricbeat' -cf ./metricbeat.tar .
tar -C 'img_helmv2' -cf ./helmv2.tar .