#!/bin/bash

echo "Cleaning diry data..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra
echo "Downloading depended Docker images..."
./download-frozen-image-v2.sh img_etcd mirrorgooglecontainers/etcd:3.2.24
./download-frozen-image-v2.sh img_hyperkube g0194776/lightning-monkey-hyperkube:v1.12.5-2
./download-frozen-image-v2.sh img_infra mirrorgooglecontainers/pause-amd64:3.1
echo "Assmebling downloaded Docker image layers to tarball files..."
tar -C 'img_etcd' -cf ./etcd.tar .
tar -C 'img_hyperkube' -cf ./k8s.tar .
tar -C 'img_infra' -cf ./infra.tar .
echo "Cleaning downloaded files..."
rm -rf img_etcd && rm -rf img_hyperkube && rm -rf img_infra