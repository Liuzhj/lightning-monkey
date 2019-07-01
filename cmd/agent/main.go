package main

import (
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"net"
	"os"
	"os/exec"
)

func main() {
	logrus.Infof("Copying depended CNI binary files...")
	cmd := exec.Command("/bin/sh", "-c", "cp -rf /tmp/cni/* /opt/cni/bin/")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		logrus.Fatalf("Failed to copy depended CNI binary files to specified OS path, error: %s", err.Error())
		return
	}
	arg := AgentArgs{}
	arg.Server = flag.String("server", "", "api address")
	arg.Address = flag.String("address", "", "local node address")
	arg.ClusterId = flag.String("cluster", "", "cluster id")
	arg.IsETCDRole = flag.Bool("etcd", false, "")
	arg.IsMasterRole = flag.Bool("master", false, "")
	arg.IsMinionRole = flag.Bool("minion", false, "")
	certdir := flag.String("cert-dir", "", "")
	flag.Parse()
	if certdir != nil && *certdir != "" {
		CERTIFICATE_STORAGE_PATH = *certdir
	}
	if arg.Server == nil || *arg.Server == "" {
		logrus.Fatalf("\"--server\" argument is required for initializing lightning-monkey agent.")
	}
	if arg.ClusterId == nil || *arg.ClusterId == "" {
		logrus.Fatalf("\"--cluster\" argument is required for initializing lightning-monkey agent.")
	}
	if !*arg.IsETCDRole && !*arg.IsMasterRole && !*arg.IsMinionRole {
		logrus.Fatalf("you must specify one role at least to initialize lightning-monkey agent.")
	}
	if arg.Address == nil || *arg.Address == "" {
		ip := GetLocalIP()
		arg.Address = &ip
	}
	agent := LightningMonkeyAgent{}
	agent.Initialize(arg)
	agent.Start()
}

type AgentArgs struct {
	AgentId      string
	Server       *string
	ClusterId    *string
	Address      *string
	LeaseId      int64
	IsETCDRole   *bool
	IsMasterRole *bool
	IsMinionRole *bool
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
