[![Build Status](https://travis-ci.com/g0194776/lightning-monkey.svg?token=vHH8ZAATWuPpZvD2YF3L&branch=feature/init)](https://travis-ci.com/g0194776/lightning-monkey)
[![codecov](https://codecov.io/gh/g0194776/lightning-monkey/branch/master/graph/badge.svg)](https://codecov.io/gh/g0194776/lightning-monkey)
![size](https://img.shields.io/github/languages/code-size/g0194776/lightning-monkey)
![go](https://img.shields.io/badge/golang-1.13%2B-blue)
![platform](https://img.shields.io/badge/platform-linux--amd64%20%7C%20darwin-lightgrey)

# Lightning Monkey
This project is currently in early development stage. Bug reports and pull requests are welcome.

# Preface
The project Lightning Monkey is a key solution for helping you deploy an entire Kubernetes cluster as well as you expected. Typically, It's a core capability of PaaS(Platform as a Service). Rather than others technological solution(Such as Rancher), It has been designed to offer more flexibility for easier integrating to yours projects.

It are consist of 2 main components:

- API Server
- Agent

The API server has responsibility for managing & scheduling deployment resource during its whole lifecycle. It'll provide a group of RESTful APIs for representing those of capabilities what described above.

The agent is a background running service which has responsibility for interacting with local resource. such as generating certificates or shipping deployment metadata etc... And... the most important thing is that the Docker runtime is required before you go.


# Supported Kubernetes Version
|Kubernetes Version|Supported|Tested|
|---|---|---|
|v1.12.5|√|√|
|v1.13.10|√|√|

# Supported Components Deployment
|Component|Version|Tested|
|---|---|---|
|ETCD|3.2.24|√|
|CoreDNS|1.5.2|√|
|Traefik|1.7.14|√|
|Kube-Router|0.2.5|√|
|Prometheus|2.2.1|√|
|HAProxy|1.7.11|√|
|KeepAlived|1.3.9|√|


# Usage

## Run API Server
```shell
#STEP 1, you need to start up an ETCD instance.
docker run -d --name etcd-server \
    --publish 2379:2379 \
    --publish 2380:2380 \
    --env ALLOW_NONE_AUTHENTICATION=yes \
    --env ETCD_ADVERTISE_CLIENT_URLS=http://etcd-server:2379 \
    bitnami/etcd:latest

#STEP 2, run API-Server.
docker run --rm -p 8080:8080 -it \
    -e "BACKEND_STORAGE_ARGS=ENDPOINTS=http://YOUR-ETCD-CLUSTER-IP:2379;LOG_LEVEL=debug" 
    lm-apiserver:201907051406
```


## Run Agent
```shell
export API_SERVER_ADDR=http://127.0.0.1:8080
export CLUSTER_ID=xxxxxxx
docker run -itd --restart=always --net=host \
    -v /etc:/etc \
    -v /var/run:/var/run \
    -v /var/lib:/var/lib \
    -v /opt/cni/bin:/opt/cni/bin \
    -e "LOG_LEVEL=debug" \
    --entrypoint=/opt/lm-agent \
    g0194776/lmagent:v0.1-8 \
      --server=$API_SERVER_ADDR \
      --address=$(ip addr show dev eth1 | grep "inet " | awk '{print $2}' | cut -f1 -d '/') \
      --cluster=$CLUSTER_ID \
      --etcd \
      --master \
      --cert-dir=/etc/kubernetes/pki
```

## Clean Up

```bash
docker ps -aq | xargs -ti docker rm -f {}
rm -rf /etc/kubernetes/
rm -rf /opt/lightning-monkey/
rm -rf /etc/cni/net.d/10-kuberouter.conf
rm -rf /var/lib/kube*
```

The details of how to run up an entire cluster using Vagrant , Please directly run this command at the root path:
```bash
vagrant up
```
It'll create & start 6 VMs to build the cluster of HA.

|VM Name|Role|
|---|---|
|k8s_master1|Kubernetes Master|
|k8s_master2|Kubernetes Master|
|k8s_master3|Kubernetes Master|
|k8s_minion1|Kubernetes Minion|
|k8s_lb1|HAProxy & KeepAlived|
|k8s_lb2|HAProxy & KeepAlived|
