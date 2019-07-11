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
    v.cpus = 3
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
            --link etcd-server:etcd-server \
            -e "BACKEND_STORAGE_ARGS=ENDPOINTS=http://etcd-server:2379;LOG_LEVEL=debug" \
            g0194776/lightning-monkey-apiserver:latest
        SHELL
        }
    end
  end
end
