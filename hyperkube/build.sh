#!/bin/bash

SUPPORTED_K8S_VERSIONS=("1.12.10" "1.13.12" "1.14.10" "1.15.9")

for ver in ${SUPPORTED_K8S_VERSIONS[*]} ; do
    echo "Building hyperkube(${ver}) image..."
    docker build -f Dockerfile.hyperkube --build-arg VERSION=${ver} -t g0194776/lightning-monkey-hyperkube:v${ver} .
    docker push g0194776/lightning-monkey-hyperkube:v${ver}
done
