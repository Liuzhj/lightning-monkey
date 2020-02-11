#!/bin/bash

echo "Downloading specified versions of kubeadm..."
arr=$1
for ver in ${arr[*]} ; do
    mkdir ${ver}
    #wget -v https://dl.k8s.io/v${ver}/kubernetes-node-linux-amd64.tar.gz -o ./${ver}/kubernetes-node-linux-amd64.tar.gz
    wget -v -O ./${ver}/kubernetes-node-linux-amd64.tar.gz https://storage.googleapis.com/kubernetes-release/release/v${ver}/kubernetes-node-linux-amd64.tar.gz
    tar -zxvf ./${ver}/kubernetes-node-linux-amd64.tar.gz -C ./${ver}
done