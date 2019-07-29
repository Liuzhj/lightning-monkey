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
        systemctl stop firewalld
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
            -e "LOG_LEVEL=debug" \
            g0194776/lightning-monkey-apiserver:latest
        sleep 5s
        echo "try to retrieving API-Server logs..."
        docker logs apiserver
        echo "preparing to create a new cluster..."
        curl -v -s -H "Content-Type: application/json" http://localhost:8080/apis/v1/cluster/create -X POST -d '{\"id\":\"1b8624d9-b3cf-41a3-a95b-748277484ba5\",\"name\":\"cluster1\",\"expected_etcd_count\":3,\"pod_network_cidr\":\"55.55.0.0/12\",\"service_cidr\":\"10.254.1.1/12\",\"kubernetes_version\":\"1.12.5\",\"service_dns_domain\":\"cluster.local\",\"network_stack\":{\"type\":\"kuberouter\"}}'
        SHELL
        }
    end
  end


  config.vm.define "k8s_master1" do |k8s_master1|
    k8s_master1.vm.box = "centos/7"
    k8s_master1.vm.network "private_network", ip: "192.168.33.11"
    k8s_master1.vm.hostname = "192.168.33.11"
    k8s_master1.trigger.after :up do |trigger|
      trigger.run_remote = {inline: <<-SHELL
        setenforce 0 && swapoff -a
        systemctl stop firewalld
        yum update -y && yum install docker -y
        sudo su && systemctl start docker && systemctl status docker
        docker run -itd --restart=always --net=host \
            --name agent \
            -v /etc:/etc \
            -v /var/run:/var/run \
            -v /var/lib:/var/lib \
            -v /opt/cni/bin:/opt/cni/bin \
            -e "LOG_LEVEL=debug" \
            --entrypoint=/opt/lm-agent \
            g0194776/lightning-monkey-agent:latest \
                --server=http://192.168.33.10:8080 \
                --address=$(ip addr show dev eth1 | grep "inet " | awk '{print $2}' | cut -f1 -d '/') \
                --cluster=1b8624d9-b3cf-41a3-a95b-748277484ba5 \
                --etcd \
                --master \
                --cert-dir=/etc/kubernetes/pki
        echo "waiting 10s..."
        sleep 10s
        echo "try to retrieving Agent logs..."
        docker logs agent
        echo "retrieving all docker containers..."
        docker ps -a
        SHELL
        }
      end
    end


    config.vm.define "k8s_master2" do |k8s_master2|
      k8s_master2.vm.box = "centos/7"
      k8s_master2.vm.network "private_network", ip: "192.168.33.12"
      k8s_master2.vm.hostname = "192.168.33.12"
      k8s_master2.trigger.after :up do |trigger|
        trigger.run_remote = {inline: <<-SHELL
          setenforce 0 && swapoff -a
          systemctl stop firewalld
          yum update -y && yum install docker -y
          sudo su && systemctl start docker && systemctl status docker
          docker run -itd --restart=always --net=host \
              --name agent \
              -v /etc:/etc \
              -v /var/run:/var/run \
              -v /var/lib:/var/lib \
              -v /opt/cni/bin:/opt/cni/bin \
              -e "LOG_LEVEL=debug" \
              --entrypoint=/opt/lm-agent \
              g0194776/lightning-monkey-agent:latest \
                  --server=http://192.168.33.10:8080 \
                  --address=$(ip addr show dev eth1 | grep "inet " | awk '{print $2}' | cut -f1 -d '/') \
                  --cluster=1b8624d9-b3cf-41a3-a95b-748277484ba5 \
                  --etcd \
                  --master \
                  --cert-dir=/etc/kubernetes/pki
          echo "waiting 10s..."
          sleep 10s
          echo "try to retrieving Agent logs..."
          docker logs agent
          echo "retrieving all docker containers..."
          docker ps -a
          SHELL
          }
        end
      end

    config.vm.define "k8s_master3" do |k8s_master3|
      k8s_master3.vm.box = "centos/7"
      k8s_master3.vm.network "private_network", ip: "192.168.33.13"
      k8s_master3.vm.hostname = "192.168.33.13"
      k8s_master3.trigger.after :up do |trigger|
        trigger.run_remote = {inline: <<-SHELL
          setenforce 0 && swapoff -a
          systemctl stop firewalld
          yum update -y && yum install docker -y
          sudo su && systemctl start docker && systemctl status docker
          docker run -itd --restart=always --net=host \
              --name agent \
              -v /etc:/etc \
              -v /var/run:/var/run \
              -v /var/lib:/var/lib \
              -v /opt/cni/bin:/opt/cni/bin \
              -e "LOG_LEVEL=debug" \
              --entrypoint=/opt/lm-agent \
              g0194776/lightning-monkey-agent:latest \
                  --server=http://192.168.33.10:8080 \
                  --address=$(ip addr show dev eth1 | grep "inet " | awk '{print $2}' | cut -f1 -d '/') \
                  --cluster=1b8624d9-b3cf-41a3-a95b-748277484ba5 \
                  --etcd \
                  --master \
                  --cert-dir=/etc/kubernetes/pki
          echo "waiting 10s..."
          sleep 10s
          echo "try to retrieving Agent logs..."
          docker logs agent
          echo "retrieving all docker containers..."
          docker ps -a
          SHELL
          }
        end
      end



      config.vm.define "k8s_minion1" do |k8s_minion1|
        k8s_minion1.vm.box = "centos/7"
        k8s_minion1.vm.network "private_network", ip: "192.168.33.14"
        k8s_minion1.vm.hostname = "192.168.33.14"
        k8s_minion1.trigger.after :up do |trigger|
          trigger.run_remote = {inline: <<-SHELL
            setenforce 0 && swapoff -a
            systemctl stop firewalld
            yum update -y && yum install docker -y
            sudo su && systemctl start docker && systemctl status docker
            docker run -itd --restart=always --net=host \
                --name agent \
                -v /etc:/etc \
                -v /var/run:/var/run \
                -v /var/lib:/var/lib \
                -v /opt/cni/bin:/opt/cni/bin \
                -e "LOG_LEVEL=debug" \
                --entrypoint=/opt/lm-agent \
                g0194776/lightning-monkey-agent:latest \
                    --server=http://192.168.33.10:8080 \
                    --address=$(ip addr show dev eth1 | grep "inet " | awk '{print $2}' | cut -f1 -d '/') \
                    --cluster=1b8624d9-b3cf-41a3-a95b-748277484ba5 \
                    --minion \
                    --cert-dir=/etc/kubernetes/pki
            echo "waiting 10s..."
            sleep 10s
            echo "try to retrieving Agent logs..."
            docker logs agent
            echo "retrieving all docker containers..."
            docker ps -a
            SHELL
            }
          end
        end
end
