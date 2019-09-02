package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	haproxy_payload = `global
    daemon
    maxconn 30000
    log 127.0.0.1 local0 debug

#---------------------------------------------------------------------
# common defaults that all the 'listen' and 'backend' sections will
# use if not designated in their block
#---------------------------------------------------------------------
defaults
    mode                    http
    log                     global
    option                  tcplog
    option                  dontlognull
    option http-server-close
    option forwardfor       except 127.0.0.0/8
    option                  redispatch
    retries                 3
    timeout http-request    10s
    timeout queue           1m
    timeout connect         10s
    timeout client          1m
    timeout server          1m
    timeout http-keep-alive 10s
    timeout check           10s
    maxconn                 30000


frontend kubernetes-apiserver
    mode                 tcp
    bind                 *:6443
    option               tcplog
    default_backend      kubernetes-apiserver

#---------------------------------------------------------------------
# round robin balancing between the various backends
#---------------------------------------------------------------------
backend kubernetes-apiserver
    mode        tcp
    balance     roundrobin
{{.MASTERS}}


listen stats
    bind *:1080
    mode http
    stats refresh 30s
    stats uri /stats`
	keepalived_payload = `vrrp_script chk_haproxy {
    script "/usr/local/bin/chk_haproxy.sh"
    interval 3
    weight 20
    rise 3
    fall 3
}

vrrp_instance VI_1 {
    state {{.STATE}}
    interface {{.ETH}}
    virtual_router_id {{.ROUTERID}}
    priority {{.PRIORITY}}
    advert_int 3                          
    track_interface {
        {{.ETH}}
    }
    #设定单播有很多好处，比如很多网络是不允许VRRP多播的
    #而设定固定范围的话就可以越过这个限制
    unicast_src_ip {{.LOCALIP}} 
    unicast_peer {
{{.MASTERIPS}}
    }
    virtual_ipaddress {
        {{.VIP}}
    }
    track_script {
        chk_haproxy
    }
}`
	keepAlivedConfigPath = "/etc/lightning-monkey/keepalived.conf"
	haProxyConfigPath    = "/etc/lightning-monkey/haproxy.cfg"
)

func HandleDeployHA(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	if job.Arguments["master-addresses"] == "" {
		return false, xerrors.Errorf("Illegal HAProxy&KeepAlived deployment job, required arguments are missed %w", crashError)
	}
	//STEP 1, generates configuration files.
	masterIPs := strings.Split(job.Arguments["master-addresses"], ",")
	haIPs := strings.Split(job.Arguments["ha-addresses"], ",")
	result, err := writeKeepAlivedConfigFile(haIPs, job, a)
	if !result || err != nil {
		return result, err
	}
	result, err = writeHAProxyConfigFile(masterIPs, job, a)
	if !result || err != nil {
		return result, err
	}
	//STEP 2, start up docker container.
	resp, err := a.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Hostname: *a.arg.Address,
		Image:    a.basicImages.Images["ha"].ImageName,
		Tty:      false,
		Volumes:  map[string]struct{}{},
	}, &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/usr/local/etc/haproxy/haproxy.cfg", haProxyConfigPath),
			fmt.Sprintf("%s:/etc/keepalived/keepalived.conf", keepAlivedConfigPath),
		},
		Privileged:    true,
		NetworkMode:   "host",
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}, &network.NetworkingConfig{}, "ha")
	if err != nil {
		return false, xerrors.Errorf("Failed to create HA container, error: %s %w", err.Error(), crashError)
	}
	if err = a.dockerClient.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return false, xerrors.Errorf("Failed to start HA container, error: %s %w", err.Error(), crashError)
	}
	out, err := a.dockerClient.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return false, xerrors.Errorf("Failed to retrieve HA container logs, error: %s %w", err.Error(), crashError)
	}
	_, _ = io.Copy(os.Stdout, out)
	return true, nil
}

func writeKeepAlivedConfigFile(haIPs []string, job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	sb := strings.Builder{}
	for i := 0; i < len(haIPs); i++ {
		if haIPs[i] != *a.arg.Address {
			sb.WriteString("        " + haIPs[i])
			if i+1 < len(haIPs) {
				sb.WriteString("\n")
			}
		}
	}
	tpl, err := template.New("kt").Parse(keepalived_payload)
	if err != nil {
		return false, fmt.Errorf("Failed to parse KeepAlived template, error: %s", err.Error())
	}
	args := map[string]string{
		"STATE":     job.Arguments["state"],
		"ETH":       *a.arg.UsedEthernetInterface,
		"ROUTERID":  job.Arguments["router-id"],
		"LOCALIP":   *a.arg.Address,
		"MASTERIPS": sb.String(),
		"VIP":       job.Arguments["vip"], //"0.0.0.0",
		"PRIORITY":  job.Arguments["priority"],
	}
	buffer := bytes.Buffer{}
	err = tpl.Execute(&buffer, args)
	if err != nil {
		return false, xerrors.Errorf("Failed to execute KeepAlived configuration template, error: %s %w", err.Error(), crashError)
	}
	conf := buffer.String()
	_ = os.Remove(keepAlivedConfigPath)
	_ = os.MkdirAll(filepath.Dir(keepAlivedConfigPath), 0644) //"rw-r-r"
	err = ioutil.WriteFile(keepAlivedConfigPath, []byte(conf), 0644)
	if err != nil {
		return false, fmt.Errorf("Failed to write KeepAlived configuration file, error: %s", err.Error())
	}
	return true, nil
}

func writeHAProxyConfigFile(masterIPs []string, job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	sb := strings.Builder{}
	for i := 0; i < len(masterIPs); i++ {
		sb.WriteString(fmt.Sprintf("    server  master-%d %s:6443 check", i, masterIPs[i]))
		if i+1 < len(masterIPs) {
			sb.WriteString("\n")
		}
	}
	tpl, err := template.New("hat").Parse(haproxy_payload)
	if err != nil {
		return false, fmt.Errorf("Failed to parse HAProxy template, error: %s", err.Error())
	}
	args := map[string]string{
		"MASTERS": sb.String(),
	}
	buffer := bytes.Buffer{}
	err = tpl.Execute(&buffer, args)
	if err != nil {
		return false, xerrors.Errorf("Failed to execute HAProxy configuration template, error: %s %w", err.Error(), crashError)
	}
	conf := buffer.String()
	_ = os.Remove(haProxyConfigPath)
	_ = os.MkdirAll(filepath.Dir(haProxyConfigPath), 0644) //"rw-r-r"
	err = ioutil.WriteFile(haProxyConfigPath, []byte(conf), 0644)
	if err != nil {
		return false, fmt.Errorf("Failed to write HAProxy configuration file, error: %s", err.Error())
	}
	return true, nil
}

func CheckHAHealth(job *entities.AgentJob, a *LightningMonkeyAgent) (bool, error) {
	var err error
	var containers []types.Container
	containers, err = a.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		logrus.Errorf("Failed to retrieve all containers information, error: %s", err.Error())
		return false, err
	}
	if containers == nil || len(containers) == 0 {
		return false, nil
	}
	for i := 0; i < len(containers); i++ {
		logrus.Debugf("container status: %s, names: %#v", containers[i].Status, containers[i].Names)
		if containers[i].Names[0] == "/ha" &&
			strings.Contains(containers[i].Status, "Up") {
			return true, nil
		}
	}
	return false, nil
}
