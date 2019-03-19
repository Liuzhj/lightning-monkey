Vagrant.configure("2") do |config|
  config.vm.box_check_update = false
  config.ssh.insert_key = false
  # set auto_update to false, if you do NOT want to check the correct
  # additions version when booting this machine
  config.vbguest.auto_update = false

  # do NOT download the iso file from a webserver
  config.vbguest.no_remote = true

  config.vm.provider "virtualbox" do |v|
    v.memory = 2048
    v.cpus = 3
  end

  config.vm.define "vm1" do |vm1|
    vm1.vm.box = "centos/7"
    vm1.vm.network "private_network", ip: "192.168.33.11"
    vm1.vm.provision "shell", inline: "setenforce 0 && swapoff -a"
    vm1.vm.hostname = "192.168.33.11"
  end

  config.vm.define "vm2" do |vm2|
    vm2.vm.box = "centos/7"
    vm2.vm.network "private_network", ip: "192.168.33.12"
    vm2.vm.provision "shell", inline: "setenforce 0 && swapoff -a"
    vm2.vm.hostname = "192.168.33.12"
  end

  config.vm.define "vm3" do |vm3|
    vm3.vm.box = "centos/7"
    vm3.vm.network "private_network", ip: "192.168.33.13"
    vm3.vm.provision "shell", inline: "setenforce 0 && swapoff -a"
    vm3.vm.hostname = "192.168.33.13"
  end
end
