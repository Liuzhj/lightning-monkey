#!/bin/bash

echo "Downloading specified versions of kubeadm..."
arr=$1
for ver in ${arr[*]} ; do
    mkdir ${ver}
    wget https://dl.k8s.io/v${ver}/kubernetes-node-linux-amd64.tar.gz -o ./${ver}/kubernetes-node-linux-amd64.tar.gz
    tar -zxvf ./${ver}/kubernetes-node-linux-amd64.tar.gz
done