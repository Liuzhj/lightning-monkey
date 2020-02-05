[![Build Status](https://travis-ci.com/g0194776/lightning-monkey.svg?token=vHH8ZAATWuPpZvD2YF3L&branch=feature/init)](https://travis-ci.com/g0194776/lightning-monkey)

# 闪电猴项目 (Lightning Monkey)
当前项目正处于开发阶段，我们欢迎任何BUG和代码的提交。

# 前言
闪电猴项目的主要定位与目标是为了解决以Kubernetes为基础的云生态底层集群自动化部署工作。在现在的社区中，如何去部署一套Kubernetes集群看起来并不难，但是如何部署一套安全的、符合企业规范，符合CNCF最佳实践规范，并且能够在私有云环境(无互联网访问权限)内进行高度自动化部署集群的技术框架是很少的。这也是闪电猴项目孕育而生的原因之一。

与其他技术方案不同的是，闪电猴项目的目标除了要完成集群的高度自动化部署之外，还做出了很多内部支持。这些内部扩展性支持将会有利于让您的项目快速集成闪电猴。也就是说，您可以把闪电猴项目当做一个单独的技术工具去运行，也可以通过API等方式快速将闪电猴项目作为整体PaaS技术方案的一部分进行快速集成。

说到部署一个Kubernetes集群，通常我们会更多的谈及Ansible技术以及如何通过一组Ansible剧本(Playbook)来完成整体部署工作。但在企业私有云环境中执行这种部署方案通常会存在一定的成本，比如如何正确管理Ansible发起主机与被控节点的互信等等，这些技术前提都将会是降低企业私有云内部主机的安全性风险。

闪电猴项目在进行Kubernetes集群部署时，除了会采用与社区中Rancher项目一样的部署模式之外，还有着很多创新的部分。

- 我们支持的第一种部署模式，类似社区中Rancher的部署模式，是需要将一条生成的命令复制下来，在被控机上进行执行，从而发起安装。这种模式的好处就是简单，并且每一台主机都可以独立执行，无需将互信权限统一交由外部一台主机进行集中管理。
- 另外一种我们支持的部署模式，我们称之为"主机管理"模式，这种模式同样需要在被控机上执行一条命令。与第一种模式不同的是，这种命令在执行时并不会为被控机指定任何部署角色(Master, Minion等等)，而是会等待闪电猴项目中的API Server进行统一调度。使用这种模式所带来的好处在企业私有云环境中显而易见，比如当前有200台主机，但是这200台主机是划分为2个集群还是划分为1个集群？这种部署规划性问题可能在主机管理层面甚至是部署集群的那一刻都是无法完全确定的。这也就意味着，采用第二种部署模式后，在API Server端将会看到一个由200台主机所组成的主机池(Pool)，集群规划者可以通过闪电猴API将任意多台池内主机划分为单独的集群，并开始触发这些节点上的部署任务。


# 已支持的Kubernetes版本 (持续增加中)
|Kubernetes Version|Supported|Tested|
|---|---|---|
|v1.12.5|√|√|
|v1.13.12|√|√|

除了支持部署Kubernetes集群之外，在闪电猴项目中，我们还支持了很多种基础组件的自动化部署任务。这些已支持的部署组件都是可选的，并且正在不断增多，列表如下:

# 已支持的部署组件列表 (持续增加中)
|Component|Version|Tested|
|---|---|---|
|ETCD|3.2.24|√|
|CoreDNS|1.5.2|√|
|Traefik|1.7.14|√|
|Kube-Router|0.2.5|√|
|Prometheus|2.2.1|√|
|HAProxy|1.7.11|√|
|KeepAlived|1.3.9|√|
|Metric-Server|0.3.3|√|
|ElasticSearch|6.8.3|√|
|Filebeat|6.8.3|√|
|Metricbeat|6.8.3|√|
|Helm v2|2.12.3|√|
|node-exporter|0.15.2|√|

如上所示，闪电猴项目不光提供了很多种可选的组件自动化部署任务，同时还会提供针对Kubernetes `kube-system` 命名空间下所有Deployment类型以及DaemonSet类型部署资源的健康检查工作，这些能力都会在闪电猴项目中以API的方式对外暴露。

# 如何使用?

## 组成部分介绍

闪电猴项目主要由3个关键组件组成:

- **ETCD**

这是闪电猴项目全局唯一的一个依赖，主要用于为API Server提供存储能力。

- **API Server**

API Server的主要职责是管理集群部署任务、动态调度部署任务，并提供一系列针对集群内已部署组件的健康检查工作等等。全局只需要部署一个API Server实例即可，一个API Server可以同时管理多个Kubernetes集群的部署任务。

- **Agent**

Agent程序需要在每一台被控机上安装，Agent的主要职责是与本地资源进行交互，比如动态生成证书、动态写入配置文件等等，它会与API Server保持心跳，用于汇报当前主机的最新状态。

## 启动API Server
```shell
docker run -d --name etcd-server \
    --publish 2379:2379 \
    --publish 2380:2380 \
    --env ALLOW_NONE_AUTHENTICATION=yes \
    --env ETCD_ADVERTISE_CLIENT_URLS=http://etcd-server:2379 \
    bitnami/etcd:latest

docker run --rm -p 8080:8080 -it \
    -e "BACKEND_STORAGE_ARGS=ENDPOINTS=http://YOUR-ETCD-CLUSTER-IP:2379;LOG_LEVEL=debug" 
    lm-apiserver:latest
```


## 启动Agent
```shell
export API_SERVER_ADDR=http://127.0.0.1:8080
export CLUSTER_ID=xxxxxxx
docker run -itd --restart=always --net=host --privileged \
    --name agent \
    -v /sys:/sys \
    -v /etc:/etc \
    -v /var/run:/var/run \
    -v /var/lib:/var/lib \
    -v /opt/cni/bin:/opt/cni/bin \
    -v /opt/lightning-monkey:/opt/lightning-monkey \
    -e "LOG_LEVEL=debug" \
    --entrypoint=/opt/lm-agent \
    g0194776/lmagent:latest \
      --server=$API_SERVER_ADDR \
      --address=$(ip addr show dev eth1 | grep "inet " | awk '{print $2}' | cut -f1 -d '/') \
      --cluster=$CLUSTER_ID \
      --etcd \
      --master \
      --cert-dir=/etc/kubernetes/pki
```

启动Agent时，是有一些启动参数的，比如当前主机的IP、所期待使用的网卡(用于绑定VIP)等等。这其中有一个参数需要特别注意，就是那个集群ID(cluster)，这个集群ID是需要率先在API Server端发起对集群的创建后，才能得到的。

在启动Agent时，针对集群ID(cluster)这个参数，您有两种选择:

- 填写一个真实的集群ID，但这个集群必须先要在闪电猴API Server中进行创建
- 填写成固定值: "00000000-0000-0000-0000-000000000000"，这等同于告诉API Server当前要注册的Agent实例是属于池化资源的


## 如何通过API Server创建一个集群

这里我们所谈到的创建一个集群，其实是创建一个集群的描述，并不是真正的去部署一个集群。这种描述是一段基于JSON格式的内容，用于详细给出待部署集群的一些内部参数，比如所使用内部域名、最少需要的Master节点数量，是否要部署HA节点等等，比如一个示例如下:

```json
{
	"id": "1b8624d9-b3cf-41a3-a95b-748277484ba5",
	"name": "demo",
	"expected_etcd_count": 3,
	"pod_network_cidr": "55.55.0.0/12",
	"service_cidr": "10.254.1.1/12",
	"kubernetes_version": "1.13.12",
	"service_dns_domain": "cluster.local",
	"service_dns_cluster_ip": "10.254.210.250",
	"ha_settings": {
		"vip": "192.168.33.100",
		"router_id": "40",
		"count": 2
	},
	"network_stack": {
		"type": "kuberouter"
	},
	"dns_deployment_settings": {
		"type": "coredns"
	},
	"node_port_range_settings": {
		"begin": 10000,
		"end": 60000
	},
	"resource_reservation": {
		"kube": "cpu=200m,memory=400Mi",
		"system": "cpu=200m,memory=400Mi"
	},
	"image_pull_secrets": [{
		"name": "myregistrykey",
		"namespace": "default",
		"registry": "repository.test.com:8444",
		"username": "admin",
		"password": "admin123",
		"email": "test@test.com"
	}],
	"ext_deployments": {
		"prometheus": {},
		"metric-server": {},
		"traefik": {},
		"es": {},
		"metricbeat": {},
		"filebeat": {},
		"helm": {}
	}
}
```

在上述所给出的JSON示例中，只有 `ext_deployments` 和 `ha_settings` 这两个节点是可选的，其余都是必选节点。而 `id` 字段也是可以不填写的，在上述给出的JSON中指定 `id` 字段主要是出于调试目的。

# 如何保证集群的HA?

通过向闪电猴API Server提交一个待部署集群的描述任务不难看出，在这个以JSON来描述的集群任务中具备一些特殊意义的字段，这些特殊意义的字段会被闪电猴API Server内部记录下来，并在具备指定条件下完成集群HA的部署工作。

|字段名称|描述|
|---|---|
|expected_etcd_count|最少待部署的ETCD节点数量，此值必须为奇数，当等于1时，部署Kubernetes集群所使用的ETCD则为单点。等于或者大于3时，闪电猴API Server会等待至少出现这些数量的节点Agent都处于运行状态，然后才会下发部署任务，让这些ETCD节点组成一个集群。需要特殊说明的是，在闪电猴项目中Kubernetes Master角色和ETCD角色是捆绑部署模式，也就是一个节点如果被设置为Master角色，那么在这个节点上会同时部署Kubernetes Master组件以及ETCD组件。|
|ha_settings.count|最少待部署的HAProxy + KeepAlived节点数量，当ha_settings节点出现，但是count为1时，代表Kubernetes Minion连接Kubernetes Master时使用虚IP，但是提供虚IP的节点只有一个。当此值大于1时，代表HAProxy + KeepAlived会被同时部署到多个节点上，并且会提供虚IP漂移的能力。|

# 如何做部署测试?

为了能够让您快速对闪电猴项目有所了解，我们为项目内部加入了基于Vagrant的自动化部署能力。您需要找一台至少具备如下资源的主机(Linux或者Mac均可)，并且安装了Vagrant + VirtualBox，就可以通过如下命令快速启动一组虚拟机，并使用闪电猴对这些虚拟机进行Kubernetes集群的安装工作。

## Kubernetes集群的创建描述(JSON)

在基于Vagrant动态创建的虚拟机环境中，其会通过闪电猴API动态提交了一个基于JSON格式的Kubernetes部署任务，在这个部署任务内，将会详细描述要部署的集群规模、是否需要HA节点、最少需要的Master节点数量以及选择安装哪些组件等等，如下:

```json
{
	"id": "1b8624d9-b3cf-41a3-a95b-748277484ba5",
	"name": "cluster1",
	"expected_etcd_count": 3,
	"pod_network_cidr": "55.55.0.0/12",
	"service_cidr": "10.254.1.1/12",
	"kubernetes_version": "1.13.10",
	"service_dns_domain": "cluster.local",
	"service_dns_cluster_ip": "10.254.210.250",
	"ha_settings": {
		"vip": "192.168.33.100",
		"router_id": "40",
		"count": 2
	},
	"network_stack": {
		"type": "kuberouter"
	},
	"dns_deployment_settings": {
		"type": "coredns"
	},
	"ext_deployments": {
		"prometheus": {},
		"metric-server": {},
		"traefik": {}
	}
}
```

## 资源需求

- CPU: 8~16(vCores)
- 内存: 至少保证有50GB可用

## 启动脚本

```bash
vagrant up
```

## 虚拟机

在成功执行上述命令后，Vagrant将会通过VirtualBox动态创建11台CentOS 7系统的主机，并在上面进行Kubernetes集群以及已选组件的自动化部署工作。

|VM Name|Role|IP|
|---|---|---|
|闪电猴API Server|n/a|192.168.33.10|
|k8s_master1|Kubernetes Master + ETCD|192.168.33.11|
|k8s_master2|Kubernetes Master + ETCD|192.168.33.12|
|k8s_master3|Kubernetes Master + ETCD|192.168.33.13
|k8s_lb1|HAProxy & KeepAlived|192.168.33.15|
|k8s_lb2|HAProxy & KeepAlived|192.168.33.16|
|k8s_minion1|Kubernetes Minion|192.168.33.20|
|k8s_minion2|Kubernetes Minion|192.168.33.21|
|k8s_minion3|Kubernetes Minion|192.168.33.22|
|k8s_minion4|Kubernetes Minion|192.168.33.23|
|k8s_minion5|Kubernetes Minion|192.168.33.24|
