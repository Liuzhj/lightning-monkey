#!/bin/bash

echo "Cleaning diry data..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra
echo "Downloading RPM packages..."
wget https://yum.dockerproject.org/repo/main/centos/7/Packages/docker-engine-1.12.6-1.el7.centos.x86_64.rpm
wget https://yum.dockerproject.org/repo/main/centos/7/Packages/docker-engine-selinux-1.12.6-1.el7.centos.noarch.rpm
echo "Downloading depended Docker images..."
./download-frozen-image-v2.sh img_etcd mirrorgooglecontainers/etcd:3.2.24
./download-frozen-image-v2.sh img_hyperkube g0194776/lightning-monkey-hyperkube:v1.13.8
./download-frozen-image-v2.sh img_infra mirrorgooglecontainers/pause-amd64:3.1
./download-frozen-image-v2.sh img_coredns coredns/coredns:1.5.2
./download-frozen-image-v2.sh img_ha pelin/haproxy-keepalived
echo "Assmebling downloaded Docker image layers to tarball files..."
tar -C 'img_etcd' -cf ./etcd.tar .
tar -C 'img_hyperkube' -cf ./k8s.tar .
tar -C 'img_infra' -cf ./infra.tar .
tar -C 'img_coredns' -cf ./coredns.tar .
tar -C 'img_ha' -cf ./ha.tar .
echo "Cleaning downloaded files..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra && rm -rf img_coredns && rm -rf img_ha