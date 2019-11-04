#!/usr/bin/env bash


#
# check and setup kubernetes env
# example: /bin/bash k8setup.sh show
# mail: chenkuo@gridsum.com
# 

VERSION=0.0.1
#LOG="/tmp/k8s_setup.log"
#APISERVER="http://192.168.56.101:8000"
#CLUSTERID="1b8624d9-b3cf-41a3-a95b-748277484ba5"

DOCKERGRAPH="/var/lib/docker"
TOKEN="df1733d40a31"

#KERNEL_URL="/pkg/kernel-ml-4.15.6-1.el7.elrepo.x86_64.rpm"
KERNEL_URL="/apis/v1/registry/software/kernel-ml-4.15.6-1.el7.elrepo.x86_64.rpm?token=${TOKEN}"
KERNEL_PKG="kernel-ml-4.15.6-1.el7.elrepo.x86_64.rpm"

DOCKER_ENGINE_URL="/apis/v1/registry/software/docker-engine-1.12.6-1.el7.centos.x86_64.rpm?token=${TOKEN}"
#DOCKER_ENGINE_URL="/pkg/docker-engine-1.12.6-1.el7.centos.x86_64.rpm"
DOCKER_ENGINE_PKG="docker-engine-1.12.6-1.el7.centos.x86_64.rpm"

DOCKER_ENGINE_SELINUX_URL="/apis/v1/registry/software/docker-engine-selinux-1.12.6-1.el7.centos.noarch.rpm?token=${TOKEN}"
#DOCKER_ENGINE_SELINUX_URL="/pkg/docker-engine-selinux-1.12.6-1.el7.centos.noarch.rpm"
DOCKER_ENGINE_SELINUX_PKG="docker-engine-selinux-1.12.6-1.el7.centos.noarch.rpm"

LMAGENT_URL="/apis/v1/registry/1.13.12/lmagent.tar?token=${TOKEN}"
#LMAGENT_URL="/pkg/lmagent.tar"
PROM_NODE_URL="/apis/v1/registry/software/exporter.tar?token=${TOKEN}"
#PROM_NODE_URL="/pkg/exporter.tar"
HELM_URL="/apis/v1/registry/software/helm-v2.12.3-linux-amd64.tar.gz?token=${TOKEN}"

_usage(){
  cat <<-EOF


  Usage: k8setup.sh [option] [command]


  Option:

    -v, --version             output program version
    -h, --help                output help information
    -e, --nic                 server nic,ex:eth1
    -a, --apiserver           apiserver url, ex:http://192.168.56.101:8080
    -g, --graph               docker data directory, ex:/data/docker
    -c, --clusterid           cluster id,ex:1b8624d9-b3cf-41a3-a95b-748277484ba5
    -r, --role                server role,support :master|minion|ha|etcd.ex:master
    -f, --force               force install,ignore kernel version,support true|false


  Command:

    run                       deployment lightningmonkey role
    check                     only check the system environment
    setup                     check and setup the system environment
    show                      show system information


  Example:

    #local run
    /bin/bash init.sh -e enp0s8 -a http://192.168.56.101:8080 -g /data/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5" -r master  -r etcd run
    /bin/bash init.sh -e enp0s8 check
    /bin/bash init.sh -e enp0s8 -a http://192.168.56.101:8080 -g /data/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5" -r master  -r etcd setup
    /bin/bash init.sh show
	
    #remote run
    #setup k8s env
    curl -fsSL http://192.168.56.101:8000/init.sh | /bin/bash /dev/stdin -a http://192.168.56.101:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5" -e enp0s8 -r master -r etcd setup
    #show server information
    curl -fsSL http://192.168.56.101:8000/init.sh | /bin/bash /dev/stdin -a http://192.168.56.101:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5" -e enp0s8 -r master -r etcd show
    #check server env
    curl -fsSL http://192.168.56.101:8000/init.sh | /bin/bash /dev/stdin -a http://192.168.56.101:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5" -e enp0s8 -r master -r etcd check
    #deploy lightingmonkey role
    curl -fsSL http://192.168.56.101:8000/init.sh | /bin/bash /dev/stdin -a http://192.168.56.101:8080 -g /var/lib/docker -c "1b8624d9-b3cf-41a3-a95b-748277484ba5" -e enp0s8 -r master -r etcd run

EOF
  exit 1
}


_version() {
  echo $VERSION
}

abort() {
  echo
  echo -e "\e[91m  $*\033[0m" 1>&2
  echo
  exit 1
}

function state() {
  local msg=$1
  local flag=$2
  if [[ "$flag" -eq 0 ]]; then
      echo -e "\e[92m PASS \033[0m $msg"
  elif [[ "$flag" -eq 2 ]]; then
      echo -e "\e[93m WARN \033[0m $msg"
  else
      echo -e "\e[91m FAIL \033[0m $msg"
  fi
}


_show_header(){
  echo
  echo -e "\e[92m  $*\033[0m" 1>&2
  echo "------------------"
}


_show_label() {
  printf "%-${SYSINFO_TITLE_WIDTH}s %s\n" "$1:" "$2"
}

_show_product() {
  _show_label "Product" "$(dmidecode -t system |awk  -F ":"  '/Product Name/{print $NF}')"
}

_show_os() {
  local distname,distid
  distname=$(awk -F '=' '/^ID=/{print $2}' /etc/os-release|tr -d "\"" )
  distid=$(awk -F '=' '/^VERSION_ID=/{print $2}' /etc/os-release |tr -d "\"" )
  _show_label "Distro" "$distname$distid"
  _show_label "Kernel" "$(uname -r)"
}

_show_cpu_and_ram() {
    local cpu
    cpu=$(grep -m1 "^model name" /proc/cpuinfo | cut -d' ' -f3- | sed -e 's/(R)//' -e 's/Core(TM) //' -e 's/CPU //')
    _show_label "CPUs" "$(nproc)x $cpu"
    _show_label "RAM" "$(awk '/^MemTotal:/ { printf "%.2f", $2/1024/1024 ; exit}' /proc/meminfo)G"
}

_show_disks() {
  for d in $(find /dev/{sd?,xvd?} -type b 2>/dev/null | sort); do
    fdisk -l "$d" 2>/dev/null |awk -F ',' '/Disk \/dev\//{print $1}'
  done
}

_show_time() {
  local timezone
  timezone=$(date +"%Z %z")
  timezone=${timezone:-UTC}
  _show_label "Timezone" "$timezone"
  _show_label "DateTime" "$(date)"
}

_show_network() {
  _show_label "DNS server" "$(more /etc/resolv.conf |awk '/nameserver/{printf "%s  ",$2}')"
  echo "******************"
  echo "Network Interface:"
  while read line; do ip a s dev "$line"|awk '/inet /{print "'"$line"' " $2}'|grep -v ^$; done< <(awk -F ":" '{if(NR>2)print $1}' /proc/net/dev)
  echo "******************"
}

_show_internet() {
  if ping -W1 -c1 114.114.114.114 &>/dev/null; then
    _show_label "Internet" "Yes"
  else
    _show_label "Internet" "No"
  fi
}


_show_main(){
  _show_header "System information"
  _show_product
  _show_os
  _show_cpu_and_ram
  _show_disks
  _show_time
  _show_internet
  _show_network
}

_check_os() {
  local distname distid
  distname=$(awk -F '=' '/^ID=/{print $2}' /etc/os-release|tr -d "\"")
  distid=$(awk -F '=' '/^VERSION_ID=/{print $2}' /etc/os-release|tr -d "\"")
  if [[ $distname == "centos" ]];then
    check_ret=0
  elif [[ $distid -ne 7 ]];then
    check_ret=1
  fi

  state "Centos7 system version" ${check_ret}
}

_check_hostname() {
  local nic=$1
  local msg="Host name"
  _hostname=$(ip a s dev "${nic}"|awk -F '[ /]+' '/inet /{print $3;exit;}')
  if [[ $(hostname) != "${_hostname}" ]];then
    state "${msg}. Suggest:${_hostname}. Actual:${HOSTNAME}" 1;return 1
  else
    state "${msg}." 0;return 0
  fi
}


_check_kernel() {
  local kernel
  local major
  local msg="Kernel version"
  kernel=$(uname  -r |awk -F '.' '{print $1}')
  major=$(uname  -r |awk -F '.' '{print $2}')
  if [[ $kernel -lt 4 ]];then
    state "${msg}. Actual:${kernel}.${major}" 1;return 1
  elif [[ $kernel -eq 4 ]] && [[ $major -lt 15 ]];then
    state "${msg}. Actual:${kernel}.${major}" 1;return 1
  else
    state "${msg}." 0;return 0
  fi
}

_check_tools() {
  if rpm -q strace lsof net-tools ipvsadm rpcbind nfs-utils wget sysstat >/dev/null 2>&1;then
    state "package strace lsof net-tools ipvsadm rpcbind nfs-utils wget sysstat " ${check_ret} return 0
  else
    state "package not install. suggest install strace lsof net-tools ipvsadm rpcbind nfs-utils wget sysstat" 1; return 1
  fi
}

_check_swap() {
  local msg="Swap off"
  local swaptotal
  swaptotal=$(free |awk '/Swap/{print $2}')
  if [[ $swaptotal -eq 0 ]];then
    state "${msg}" 0;return 0
  else
    state "${msg}. Actual:Enabled" 1;return 1
  fi
}

_check_limit() {
  local maxfile
  local maxproc
  local flag=0
  maxfile=$(ulimit -n)
  maxproc=$(ulimit -u)
  if [[ ${maxfile} -ge 655360 ]];then
    state "Limit max file." 0
  else
    state "Limit max file. Suggest:655360. Actual:${maxfile}" 1;flag=1
  fi

  if [[ ${maxproc} -ge 131072 ]];then
    state "Limit user processes" 0
  else
    state "Limit user processes. Suggest:131072. Actual:${maxproc}" 1;flag=1
  fi
  return ${flag}
}

_check_sysctl() {
  local flag=0
  conntrack_max=$(sysctl -n net.netfilter.nf_conntrack_max 2>/dev/null)
  ip_forward=$(sysctl -n net.ipv4.ip_forward 2>/dev/null)
  swappiness=$(sysctl -n vm.swappiness 2>/dev/null)
  map_count=$(sysctl -n vm.max_map_count 2>/dev/null)
  pid_max=$(sysctl -n kernel.pid_max 2>/dev/null)
  max_user_watches=$(sysctl -n fs.inotify.max_user_watches 2>/dev/null)
  max_user_instances=$(sysctl -n fs.inotify.max_user_instances 2>/dev/null)
  ip_nonlocal_bind=$(sysctl -n net.ipv4.ip_nonlocal_bind 2>/dev/null)

  if [[ $conntrack_max -lt 655360 ]];then
    state "kernel conntrack_max. Suggest:655360. Actual:${conntrack_max}" 1;flag=1
  else
    state "kernel conntrack_max. Suggest:655360" 0
  fi

  if [[ $ip_forward -ne 1 ]];then 
    state "kernel ip_forward. Suggest:1. Actual:${ip_forward}" 1;flag=1
  else
    state "kernel kernel ip_forward" 0
  fi
  
  if [[ $swappiness -ne 0 ]];then
    state "kernel swappiness. Suggest:0. Actual:${swappiness}" 1;flag=1
  else
    state "kernel kernel swappiness" 0
  fi

  if [[ $map_count -lt 262144 ]];then
    state "kernel map_count. Suggest:262144. Actual:${map_count}" 2;flag=1
  else
    state "kernel kernel map_count" 0
  fi

  if [[ $pid_max -lt 4194303 ]];then
    state "kernel pid_max. Suggest:4194303. Actual:${pid_max}" 2;flag=1
  else
    state "kernel pid_max" 0
  fi

  if [[ $max_user_watches -lt 1048576 ]];then
    state "kernel max_user_watches.Suggest:1048576. Actual:${max_user_watches}" 2;flag=1
  else
    state "kernel max_user_watches" 0
  fi

  if [[ $max_user_instances -lt 8192 ]];then
    state "kernel max_user_instances.Suggest:8192. Actual:${max_user_instances}" 2;flag=1
  else
    state "kernel kernel max_user_instances" 0
  fi

  if [[ $ip_nonlocal_bind -ne 1 ]];then
    state "kernel ip_nonlocal_bind.Suggest:1. Actual:${ip_nonlocal_bind}" 1;flag=1
  else
    state "kernel kernel ip_nonlocal_bind" 0
  fi

  if [[ "${flag}" == 1 ]];then
    return 1
  else
    return 0
  fi
}

_check_timezone() {
  local timezone
  local msg="Time zone +0800"
  timezone=$(date +"%z")
  if [[ ${timezone} == "+0800" ]];then
    state "${msg}" 0 && return 0
  else
    state "${msg}. Actual: ${timezone}" 2 && return 1
  fi
}

_check_ntp() {
  local msg="chronyd enabled"
  if chronyc -a makestep > /dev/null 2>&1;then
    state "${msg}" 0 && return 0
  else
    state "${msg}. Actual: Disabled" 2 && return 1
  fi
}

_check_selinux() {
  local msg="SELinux disabled"
  case $(getenforce) in
      Disabled|Permissive) state "${msg}" 0 && return 0;;
      *)                   state "${msg}. Actual: $(getenforce)" 1 && return 1;;
  esac
}

_check_firewalld() {
  local msg="firewalld disabled"
  if ! systemctl status firewalld >/dev/null 2>&1;then
    state "${msg}" 0 && return 0
  else
    state "${msg}. Actual: Running" 1 && return 1
  fi
}

_check_docker() {
  local version
  local msg="Docker version"
  version=$(docker -v 2>/dev/null|awk '{print $3}'|tr -d ",")
  
  if [[ ${version} == "1.12.6" ]];then
    state "${msg}" 0;return 0
  else
    state "${msg}" 1;return 1
  fi
}

_check_kernel_module() {
  msg="ip conntrack module"
  if lsmod|grep nf_conntrack_ipv4>/dev/null 2>&1;then
    state "${msg}" 0;return 0
  else
    state "${msg}. Actual:not load" 1;return 1
  fi
  
}

_check_repo() {
  local httpcode
  msg="Centos Repo check"
  for repourl in $(yum repolist -v | grep Repo-baseurl | awk  '{print $3}')
  do
    httpcode=$(curl -s --head --write-out "%{http_code}\n" "${repourl}" -o  /dev/null)
    if [[ ${httpcode} == 200 ]];then
      state "${msg}" 0;return 0
    else
      state "${msg}" 1;return 1
    fi
  done
}

_check_main(){
  _show_header "Check environment"
  local nic=$1
  local apiserver=$2
  local clusterid=$3
  local graph=$4
  local role
  role=$(echo ${@:5}|sed -e 's/ / --/g' -e 's/^/ --/g')

  _check_os
  _check_repo
  _check_kernel
  _check_kernel_module
  _check_tools
  _check_swap
  _check_limit
  _check_sysctl
  _check_timezone
  _check_hostname ${nic}
  _check_ntp
  _check_selinux
  _check_firewalld
  _check_docker
}


_setup_kernel() {
  local apiserver=$1
  if ! command -v wget >/dev/null 2>&1;then
    abort "wget command not found"
  fi
  wget "${apiserver}${KERNEL_URL}" -O /tmp/${KERNEL_PKG}
  rpm -ivh /tmp/${KERNEL_PKG}
  grub2-set-default 0
  grub2-mkconfig -o /boot/grub2/grub.cfg 
}

_setup_tools() {
  yum install -y strace lsof net-tools ipvsadm rpcbind nfs-utils wget chrony sysstat
  yum install -y epel-release
  yum install -y jq
}

_setup_swap() {
  sed -i '/ swap / s/^/#/' /etc/fstab
  swapoff -a
  free -m 
}

_setup_limit() {
  echo "* - nproc 131072" >> /etc/security/limits.conf
  echo "* - nofile 655360" >> /etc/security/limits.conf
  ulimit -u 131072
  ulimit -n 655360
}

_setup_sysctl() {
  cat > /etc/sysctl.d/k8s.conf <<EOF
net.netfilter.nf_conntrack_max = 655360
net.netfilter.nf_conntrack_timestamp = 1
net.netfilter.nf_conntrack_acct = 1
net.ipv4.ip_forward = 1
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.neigh.default.gc_thresh1 = 4096
net.ipv4.neigh.default.gc_thresh2 = 6144
net.ipv4.neigh.default.gc_thresh3 = 8192
net.core.somaxconn = 1024
vm.swappiness = 0
vm.max_map_count = 262144
kernel.pid_max = 4194303
fs.inotify.max_user_watches=1048576
fs.inotify.max_user_instances = 8192
net.ipv4.ip_nonlocal_bind = 1
EOF
  sysctl -p /etc/sysctl.d/k8s.conf >/dev/null 2>&1
}


_setup_timezone() {
  ln -sfv /usr/share/zoneinfo/Asia/Shanghai /etc/localtime 
}

_setup_ntp() {
  systemctl restart chronyd 
  chronyc -a makestep
}

_setup_selinux() {
  sed -i 's/SELINUX=enforcing/SELINUX=disabled/g' /etc/selinux/config
  setenforce 0 
}

_setup_firewalld() {
  systemctl stop firewalld
  systemctl disable firewalld
}

_setup_docker() {
  local dir=$1
  local apiserver=$2
  if ! command -v wget>/dev/null 2>&1;then
    abort "wget command not found"
  fi
  yum remove -y docker*
  wget ${apiserver}${DOCKER_ENGINE_URL} -O /tmp/${DOCKER_ENGINE_PKG}
  wget ${apiserver}${DOCKER_ENGINE_SELINUX_URL} -O /tmp/${DOCKER_ENGINE_SELINUX_PKG}
  yum install -y /tmp/${DOCKER_ENGINE_PKG} /tmp/${DOCKER_ENGINE_SELINUX_PKG}

  pt="/usr/bin/dockerd --log-opt max-size=50M -H unix:///var/run/docker.sock -H tcp://0.0.0.0:900 --graph=${dir}"
  execstart=$(echo ${pt}|sed 's/\//\\\//g')
  sed -r -i 's/(^ExecStart=)[^"]*/\1'"${execstart}"'/' /usr/lib/systemd/system/docker.service

  systemctl enable docker
  systemctl restart docker

  mount -o remount,rw '/sys/fs/cgroup'
  rm -f /sys/fs/cgroup/cpuacct,cpu
  ln -sfv /sys/fs/cgroup/cpu,cpuacct /sys/fs/cgroup/cpuacct,cpu
}


_setup_kernel_module() {
  cat > /etc/sysconfig/modules/ipvs.modules <<EOF
#!/bin/bash
modprobe -- ip_vs
modprobe -- ip_vs_rr
modprobe -- ip_vs_wrr
modprobe -- ip_vs_sh
modprobe -- nf_conntrack_ipv4
modprobe -- ip_conntrack
modprobe -- br_netfilter
EOF

  chmod 755 /etc/sysconfig/modules/ipvs.modules \
    && bash /etc/sysconfig/modules/ipvs.modules \
    && lsmod | grep -e br_netfilter -e nf_conntrack_ipv4
}


_setup_hostname() {
  local nic=$1
  _hostname=$(ip a s dev "${nic}"|awk -F '[ /]+' '/inet /{print $3;exit;}')
  hostnamectl set-hostname "${_hostname}"
}

_show_setup_pass(){
  echo
  echo -e "\e[92m  $*\033[0m" 1>&2
  echo "------------------"
}

_setup_main(){

  _show_header "setup environment"

  local nic=$1
  local apiserver=$2
  local clusterid=$3
  local graph=$4
  local role
  role=$(echo ${@:5}|sed -e 's/ / --/g' -e 's/^/ --/g')

  _check_os >/dev/null 2>&1 || abort "Only supports centos7 system version." 

  _check_repo >/dev/null 2>&1 || abort "No available centos repo"

  _show_setup_pass "Install tools"
  _check_tools>/dev/null 2>&1 || _setup_tools

  _show_setup_pass "Swap off"
  _check_swap>/dev/null 2>&1 || _setup_swap

  _show_setup_pass "Tune ulimit"
  _check_limit>/dev/null 2>&1 || _setup_limit

  _show_setup_pass "Load kernel module"
  _check_kernel_module>/dev/null 2>&1 || _setup_kernel_module

  _show_setup_pass "Tune sysctl"
  _check_sysctl>/dev/null 2>&1 || _setup_sysctl
  
  _show_setup_pass "Configure timezone"
  _check_timezone>/dev/null 2>&1 || _setup_timezone

  _show_setup_pass "Enable ntp"
  _check_ntp>/dev/null 2>&1 || _setup_ntp

  _show_setup_pass "Disable selinux"
  _check_selinux>/dev/null 2>&1 || _setup_selinux

  _show_setup_pass "Disable firewalld"
  _check_firewalld>/dev/null 2>&1 || _setup_firewalld

  _show_setup_pass "Configure hostname"
  _check_hostname "${nic}">/dev/null 2>&1 || _setup_hostname "${nic}"

  _show_setup_pass "Install docker"
  _check_docker>/dev/null 2>&1 || _setup_docker "${graph}" "${apiserver}"

  _show_setup_pass "Upgrade kernel"
  _check_kernel>/dev/null 2>&1 || _setup_kernel "${apiserver}"


}

_run_main() {
  local nic=$1
  local apiserver=$2
  local clusterid=$3
  local graph=$4
  local force=$5
  local role
  role=$(echo ${@:6}|sed -e 's/ / --/g' -e 's/^/ --/g')

  if [[ "${force}" == "false" ]];then
    _check_kernel>/dev/null 2>&1 || abort "No kernel upgrade to 4.15.x"
  fi
  _check_os >/dev/null 2>&1 || abort "Only supports centos7 system version."
  _check_swap>/dev/null 2>&1 || abort "No Swap disabled"
  _check_sysctl>/dev/null 2>&1 || abort "No Kernel Parameters configured"
  _check_selinux>/dev/null 2>&1 || abort "No Selinux disabled"
  _check_firewalld>/dev/null 2>&1 || abort "No Firewalld disabled"
  _check_hostname "${nic}">/dev/null 2>&1 || abort "Host name was not configured correctly"
  _check_docker>/dev/null 2>&1 || abort "No docker install"

  if ! command -v wget >/dev/null 2>&1;then
    abort "wget command not found"
  fi
  
  #check node exporter
  #netstat -tulnp|grep 9100
  #curl http://localhost:9100/metrics
  wget "${apiserver}${PROM_NODE_URL}" -O /tmp/exporter.tar
  docker load </tmp/exporter.tar
  docker run -d --net="host" --pid="host" --cap-add=SYS_TIME prom/node-exporter:v0.15.2
  
  #deployment lmagent
  wget "${apiserver}${LMAGENT_URL}" -O /tmp/lmagent.tar
  docker load </tmp/lmagent.tar

  #install helm tools
  wget "${apiserver}${HELM_URL}" -O /tmp/helm-v2.12.3-linux-amd64.tar.gz
  tar xvzfp /tmp/helm-v2.12.3-linux-amd64.tar.gz linux-amd64/helm
  /bin/cp -f linux-amd64/helm /usr/bin/ && chmod 755 /usr/bin/helm

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
            g0194776/lightning-monkey-agent:latest \
                --server="${apiserver}" \
                --nc="${nic}" \
                --address="$(ip a s dev "${nic}"|awk -F '[ /]+' '/inet /{print $3;exit;}')" \
                --cluster="${clusterid}" \
                --cert-dir=/etc/kubernetes/pki \
                ${role}
}

if [[ $UID != 0 ]]; then
  abort "current user isn't root"
fi

if [[ $# -eq 0 ]] ; then
  _usage
fi

while test $# -ne 0; do
  arg=$1; shift
  case $arg in
    -h|--help)       _usage; exit ;;
    -v|--version)    _version; exit ;;
    -e|--nic)        nic="${1}"; shift ;;
    -r|--role)       role="${role} ${1}"; shift ;;
    -a|--apiserver)  apiserver="${1}"; shift ;;
    -c|--clusterid)  clusterid="${1}"; shift ;;
    -g|--graph)      graph="${1}"; shift ;;
    -f|--force)      force="${1}"; shift ;;
    run)             [[ -z "${nic}" || -z "${apiserver}" || -z "${clusterid}" || -z "${role}" ]] && _usage
                     [[ -z "${graph}" ]] && graph="${DOCKERGRAPH}" 
                     [[ -z "${force}" ]] && force="false"
                     _run_main "${nic}" "${apiserver}" "${clusterid}" "${graph}" "${force}" "${role}";;

    check)           [[ -z "${nic}" ]] && _usage
                     _check_main "${nic}";;

    show)            _show_main ;;

    setup)           [[ -z "${nic}" || -z "${apiserver}" || -z "${clusterid}" || -z "${role}" ]] && _usage
                     [[ -z "${graph}" ]] && graph="${DOCKERGRAPH}" 
                      _setup_main "${nic}" "${apiserver}" "${clusterid}" "${graph}" "${role}";;
                    
    *)_usage
      ;;
  esac
done

