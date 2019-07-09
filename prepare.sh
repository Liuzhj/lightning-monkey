#!/bin/bash

echo "Cleaning diry data..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra
echo "Downloading depended Docker images..."
./download-frozen-image-v2.sh img_etcd mirrorgooglecontainers/etcd:3.2.24
./download-frozen-image-v2.sh img_hyperkube g0194776/lightning-monkey-hyperkube:v1.12.5-2
./download-frozen-image-v2.sh img_infra mirrorgooglecontainers/pause-amd64:3.1
echo "Assmebling downloaded Docker image layers to tarball files..."
mkdir -p cmd/apiserver/apis/v1/registry/1.12.5
tar -C 'img_etcd' -cf cmd/apiserver/apis/v1/registry/1.12.5/etcd.tar .
tar -C 'img_hyperkube' -cf cmd/apiserver/apis/v1/registry/1.12.5/k8s.tar .
tar -C 'img_infra' -cf cmd/apiserver/apis/v1/registry/1.12.5/infra.tar .
echo "Cleaning downloaded files..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra