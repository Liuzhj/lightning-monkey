#!/bin/bash

echo "Cleaning diry data..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra && rm -rf img_coredns && rm -rf img_ha && rm -rf img_metrics && rm -rf img_traefik && rm -rf img_router && rm -rf img_busybox && rm -rf img_prometheus
echo "Downloading RPM packages..."
wget https://yum.dockerproject.org/repo/main/centos/7/Packages/docker-engine-1.12.6-1.el7.centos.x86_64.rpm
wget https://yum.dockerproject.org/repo/main/centos/7/Packages/docker-engine-selinux-1.12.6-1.el7.centos.noarch.rpm
wget http://mirror.rc.usf.edu/compute_lock/elrepo/kernel/el7/x86_64/RPMS/kernel-ml-4.15.6-1.el7.elrepo.x86_64.rpm
echo "Downloading depended Docker images..."
./download-frozen-image-v2.sh img_etcd mirrorgooglecontainers/etcd:3.2.24
./download-frozen-image-v2.sh img_hyperkube g0194776/lightning-monkey-hyperkube:v1.13.10
./download-frozen-image-v2.sh img_infra mirrorgooglecontainers/pause-amd64:3.1
./download-frozen-image-v2.sh img_coredns coredns/coredns:1.5.2
./download-frozen-image-v2.sh img_ha pelin/haproxy-keepalived:latest
./download-frozen-image-v2.sh img_metrics mirrorgooglecontainers/metrics-server-amd64:v0.3.3
./download-frozen-image-v2.sh img_traefik traefik:1.7.14
./download-frozen-image-v2.sh img_router cloudnativelabs/kube-router:v0.2.5
./download-frozen-image-v2.sh img_busybox busybox:latest
./download-frozen-image-v2.sh img_prometheus prom/prometheus:v2.2.1
echo "Assmebling downloaded Docker image layers to tarball files..."
tar -C 'img_etcd' -cf ./etcd.tar .
tar -C 'img_hyperkube' -cf ./k8s.tar .
tar -C 'img_infra' -cf ./infra.tar .
tar -C 'img_coredns' -cf ./coredns.tar .
tar -C 'img_ha' -cf ./ha.tar .
tar -C 'img_metrics' -cf ./metrics.tar .
tar -C 'img_traefik' -cf ./traefik.tar .
tar -C 'img_router' -cf ./router.tar .
tar -C 'img_busybox' -cf ./busybox.tar .
tar -C 'img_prometheus' -cf ./prometheus.tar .
echo "Cleaning downloaded files..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra && rm -rf img_coredns && rm -rf img_ha && rm -rf img_metrics && rm -rf img_traefik && rm -rf img_router && rm -rf img_busybox && rm -rf img_prometheus