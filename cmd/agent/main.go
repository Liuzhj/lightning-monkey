package main

import (
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"net"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	if runtime.GOOS == "linux" {
		logrus.Infof("Copying depended CNI binary files...")
		cmd := exec.Command("/bin/sh", "-c", "cp -rf /tmp/cni/* /opt/cni/bin/")
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			logrus.Fatalf("Failed to copy depended CNI binary files to specified OS path, error: %s", err.Error())
			return
		}
	}
	arg := AgentArgs{}
	arg.Server = flag.String("server", "", "api address")
	arg.Address = flag.String("address", "", "local node address")
	arg.UsedEthernetInterface = flag.String("nc", "", "used ethernet interface name")
	arg.ClusterId = flag.String("cluster", "", "cluster id")
	arg.NodeLabels = flag.String("labels", "", "Labels to add when registering the node in the cluster. Labels must be key=value pairs separated by ','. Labels in the 'kubernetes.io' namespace must begin with an allowed prefix (kubelet.kubernetes.io, node.kubernetes.io) or be in the specifically allowed set (beta.kubernetes.io/arch, beta.kubernetes.io/instance-type, beta.kubernetes.io/os, failure-domain.beta.kubernetes.io/region, failure-domain.beta.kubernetes.io/zone, failure-domain.kubernetes.io/region, failure-domain.kubernetes.io/zone, kubernetes.io/arch, kubernetes.io/hostname, kubernetes.io/instance-type, kubernetes.io/os)")
	arg.IsETCDRole = flag.Bool("etcd", false, "")
	arg.IsMasterRole = flag.Bool("master", false, "")
	arg.IsMinionRole = flag.Bool("minion", false, "")
	arg.IsHARole = flag.Bool("ha", false, "")
	arg.ListenPort = flag.Int("port", 6060, "The port used for listening API call.")
	id := flag.String("id", "", "Specify the fixed ID for current agent instance, that's available only for debugging.")
	certdir := flag.String("cert-dir", "", "")
	flag.Parse()
	if id != nil && *id != "" {
		arg.AgentId = *id
	}
	if certdir != nil && *certdir != "" {
		CERTIFICATE_STORAGE_PATH = *certdir
	}
	if arg.Server == nil || *arg.Server == "" {
		logrus.Fatalf("\"--server\" argument is required for initializing lightning-monkey agent.")
	}
	if arg.UsedEthernetInterface == nil || *arg.UsedEthernetInterface == "" {
		logrus.Fatalf("\"--nc\" argument is required for initializing lightning-monkey agent.")
	}
	if arg.ClusterId == nil || *arg.ClusterId == "" {
		logrus.Fatalf("\"--cluster\" argument is required for initializing lightning-monkey agent.")
	}
	if arg.Address == nil || *arg.Address == "" {
		ip := GetLocalIP()
		arg.Address = &ip
	}
	common.CertManager = &certs.CertificateManagerImple{}
	agent := LightningMonkeyAgent{}
	agent.Initialize(arg)
	go agent.Start()
	agent.InitializeWebServer()
}

type AgentArgs struct {
	AgentId               string
	Server                *string
	ClusterId             *string
	Address               *string
	NodeLabels            *string
	UsedEthernetInterface *string
	LeaseId               int64
	IsETCDRole            *bool
	IsMasterRole          *bool
	IsMinionRole          *bool
	IsHARole              *bool
	ListenPort            *int
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
