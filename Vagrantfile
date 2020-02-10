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
    v.memory = 3072
    #v.cpus = 3
  end

  config.vm.define "apiserver" do |apiserver|
    apiserver.vm.box = "centos/7"
    apiserver.vm.network "private_network", ip: "192.168.33.10"
    apiserver.vm.hostname = "192.168.33.10"
    apiserver.trigger.after :up do |trigger|
      trigger.run_remote = {inline: <<-SHELL
        setenforce 0 && swapoff -a && mkdir /opt/etcd/data -p
        systemctl stop firewalld
        yum update -y && yum install docker -y
        sudo su && systemctl start docker && systemctl status docker
        docker run -itd --name etcd-server \
           --publish 2379:2379 \
           --publish 2380:2380 \
           --env ALLOW_NONE_AUTHENTICATION=yes \
           --env ETCD_ADVERTISE_CLIENT_URLS=http://etcd-server:2379 \
           -v /opt/etcd/data:/opt/bitnami/etcd/data \
           bitnami/etcd:latest \
           etcd --name etcd-s1 \
           --auto-compaction-retention=1 --max-request-bytes=33554432 --quota-backend-bytes=8589934592
        docker run -itd -p 8080:8080 \
            --name apiserver \
            --link etcd-server:etcd-server \
            -e "BACKEND_STORAGE_ARGS=ENDPOINTS=http://etcd-server:2379;LOG_LEVEL=debug" \
            -e "LOG_LEVEL=debug" \
            registry.cn-beijing.aliyuncs.com/lightning-monkey/apiserver:latest
        sleep 5s
        echo "try to retrieving API-Server logs..."
        docker logs apiserver
        echo "preparing to create a new cluster..."
        curl -v -s -H "Content-Type: application/json" http://localhost:8080/apis/v1/cluster/create -X POST -d '{\"id\":\"1b8624d9-b3cf-41a3-a95b-748277484ba5\",\"name\":\"demo\",\"expected_etcd_count\":3,\"pod_network_cidr\":\"55.55.0.0/12\",\"service_cidr\":\"10.254.1.1/12\",\"kubernetes_version\":\"1.14.10\",\"service_dns_domain\":\"cluster.local\",\"service_dns_cluster_ip\":\"10.254.210.250\",\"ha_settings\":{\"vip\":\"192.168.33.100\",\"router_id\":\"40\",\"count\":2},\"network_stack\":{\"type\":\"kuberouter\"},\"dns_deployment_settings\":{\"type\":\"coredns\"},\"node_port_range_settings\":{\"begin\":10000,\"end\":60000},\"resource_reservation\":{\"kube\":\"cpu=200m,memory=400Mi\",\"system\":\"cpu=200m,memory=400Mi\"},\"image_pull_secrets\":[{\"name\":\"myregistrykey\",\"namespace\":\"default\",\"registry\":\"repository.test.com:8444\",\"username\":\"admin\",\"password\":\"admin123\",\"email\":\"test@test.com\"}],\"ext_deployments\":{\"prometheus\":{},\"metric-server\":{},\"traefik\":{},\"es\":{},\"metricbeat\":{},\"filebeat\":{},\"helm\":{}}}'
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
        curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r master  -r etcd setup
        curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r master  -r etcd -f true run
        echo "waiting 100s..."
        sleep 100s
        sudo docker cp agent:/usr/bin/kubectl /usr/bin
        sudo mkdir ~/.kube/ && sudo curl -s http://192.168.33.10:8080/apis/v1/certs/admin/get?cluster=1b8624d9-b3cf-41a3-a95b-748277484ba5 | jq -r .content | sed 's/\\n/\n/g' > ~/.kube/config
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
          curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r master  -r etcd setup
          curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r master  -r etcd -f true run
          echo "waiting 100s..."
          sleep 100s
          sudo docker cp agent:/usr/bin/kubectl /usr/bin
          sudo mkdir ~/.kube/ && sudo curl -s http://192.168.33.10:8080/apis/v1/certs/admin/get?cluster=1b8624d9-b3cf-41a3-a95b-748277484ba5 | jq -r .content | sed 's/\\n/\n/g' > ~/.kube/config
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
          curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r master  -r etcd setup
          curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r master  -r etcd -f true run
          echo "waiting 100s..."
          sleep 100s
          sudo docker cp agent:/usr/bin/kubectl /usr/bin
          sudo mkdir ~/.kube/ && sudo curl -s http://192.168.33.10:8080/apis/v1/certs/admin/get?cluster=1b8624d9-b3cf-41a3-a95b-748277484ba5 | jq -r .content | sed 's/\\n/\n/g' > ~/.kube/config
          echo "try to retrieving Agent logs..."
          docker logs agent
          echo "retrieving all docker containers..."
          docker ps -a
          SHELL
          }
        end
      end


      config.vm.define "k8s_lb1" do |k8s_lb1|
       k8s_lb1.vm.box = "centos/7"
       k8s_lb1.vm.network "private_network", ip: "192.168.33.15"
       k8s_lb1.vm.hostname = "192.168.33.15"
       k8s_lb1.trigger.after :up do |trigger|
         trigger.run_remote = {inline: <<-SHELL
           curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r ha setup
           curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r ha -f true run
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


      config.vm.define "k8s_lb2" do |k8s_lb2|
       k8s_lb2.vm.box = "centos/7"
       k8s_lb2.vm.network "private_network", ip: "192.168.33.16"
       k8s_lb2.vm.hostname = "192.168.33.16"
       k8s_lb2.trigger.after :up do |trigger|
         trigger.run_remote = {inline: <<-SHELL
           curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r ha setup
           curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r ha -f true run
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
        k8s_minion1.vm.network "private_network", ip: "192.168.33.20"
        k8s_minion1.vm.hostname = "192.168.33.20"
        k8s_minion1.vm.provider :virtualbox do |vb|
          vb.customize ["modifyvm", :id, "--memory", "6144"]
        end
        k8s_minion1.trigger.after :up do |trigger|
          trigger.run_remote = {inline: <<-SHELL
		        curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion setup
            curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion -f true run
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

      config.vm.define "k8s_minion2" do |k8s_minion2|
        k8s_minion2.vm.box = "centos/7"
        k8s_minion2.vm.network "private_network", ip: "192.168.33.21"
        k8s_minion2.vm.hostname = "192.168.33.21"
        k8s_minion2.vm.provider :virtualbox do |vb|
          vb.customize ["modifyvm", :id, "--memory", "6144"]
        end
        k8s_minion2.trigger.after :up do |trigger|
          trigger.run_remote = {inline: <<-SHELL
		        curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion setup
            curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion -f true run
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
       
       config.vm.define "k8s_minion3" do |k8s_minion3|
         k8s_minion3.vm.box = "centos/7"
         k8s_minion3.vm.network "private_network", ip: "192.168.33.22"
         k8s_minion3.vm.hostname = "192.168.33.22"
         k8s_minion3.vm.provider :virtualbox do |vb|
           vb.customize ["modifyvm", :id, "--memory", "6144"]
         end
         k8s_minion3.trigger.after :up do |trigger|
           trigger.run_remote = {inline: <<-SHELL
             curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion setup
             curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion -f true run
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
        
        config.vm.define "k8s_minion4" do |k8s_minion4|
          k8s_minion4.vm.box = "centos/7"
          k8s_minion4.vm.network "private_network", ip: "192.168.33.23"
          k8s_minion4.vm.hostname = "192.168.33.23"
          k8s_minion4.vm.provider :virtualbox do |vb|
            vb.customize ["modifyvm", :id, "--memory", "6144"]
          end
          k8s_minion4.trigger.after :up do |trigger|
            trigger.run_remote = {inline: <<-SHELL
              curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion setup
              curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion -f true run
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

         config.vm.define "k8s_minion5" do |k8s_minion5|
           k8s_minion5.vm.box = "centos/7"
           k8s_minion5.vm.network "private_network", ip: "192.168.33.24"
           k8s_minion5.vm.hostname = "192.168.33.24"
           k8s_minion5.vm.provider :virtualbox do |vb|
             vb.customize ["modifyvm", :id, "--memory", "6144"]
           end
           k8s_minion5.trigger.after :up do |trigger|
             trigger.run_remote = {inline: <<-SHELL
               curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion setup
               curl -fsSL http://192.168.33.10:8080/bootstrap/init.sh | /bin/bash /dev/stdin -a http://192.168.33.10:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5"  -e eth1  -r minion -f true run
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
