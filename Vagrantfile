CLUSTER_ID = "1b8624d9-b3cf-41a3-a95b-748277484ba5"

Vagrant.configure("2") do |config|
  config.vm.box_check_update = false
  config.ssh.insert_key = false
  # set auto_update to false, if you do NOT want to check the correct
  # additions version when booting this machine
  config.vbguest.auto_update = false

  # do NOT download the iso file from a webserver
  config.vbguest.no_remote = true
  # resolved long time waiting problem on CentOS 7.
  # https://github.com/devopsgroup-io/vagrant-digitalocean/issues/255
  config.vm.synced_folder '.', '/vagrant', disabled: true

  config.vm.provider "virtualbox" do |v|
    v.memory = 2048
    #v.cpus = 3
  end

  config.vm.define "apiserver" do |apiserver|
    apiserver.vm.box = "centos/7"
    apiserver.vm.network "private_network", ip: "192.168.33.10"
    apiserver.vm.hostname = "192.168.33.10"
    apiserver.trigger.after :up do |trigger|
      trigger.run_remote = {inline: <<-SHELL
        setenforce 0 && swapoff -a
        yum update -y && yum install docker -y
        sudo su && systemctl start docker && systemctl status docker
        docker run -itd --name etcd-server \
            --publish 2379:2379 \
            --publish 2380:2380 \
            --env ALLOW_NONE_AUTHENTICATION=yes \
            --env ETCD_ADVERTISE_CLIENT_URLS=http://etcd-server:2379 \
            bitnami/etcd:latest
        docker run -itd -p 8080:8080 \
            --name apiserver \
            --link etcd-server:etcd-server \
            -e "BACKEND_STORAGE_ARGS=ENDPOINTS=http://etcd-server:2379;LOG_LEVEL=debug" \
            g0194776/lightning-monkey-apiserver:latest
        sleep 5s
        echo "try to retrieving API-Server logs..."
        docker logs apiserver
        echo "preparing to create a new cluster..."
        curl -v -s -H "Content-Type: application/json" http://localhost:8080/apis/v1/cluster/create -X POST -d '{\"id\":\"1b8624d9-b3cf-41a3-a95b-748277484ba5\",\"name\":\"cluster1\",\"expected_etcd_count\":1,\"pod_network_cidr\":\"55.55.0.0/12\",\"service_cidr\":\"10.254.1.1/12\",\"kubernetes_version\":\"1.12.5\",\"service_dns_domain\":\"cluster.local\",\"network_stack\":{\"type\":\"kuberouter\"}}'
        SHELL
        }
    end
  end
end
