package main

import (
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"net"
)

func main() {
	arg := AgentArgs{}
	arg.Server = flag.String("server", "", "api address")
	arg.Address = flag.String("address", "", "local node address")
	arg.MetadataId = flag.String("metadata", "", "metadata id")
	arg.ClusterId = flag.String("cluster", "", "cluster id")
	arg.IsETCDRole = flag.Bool("etcd", false, "")
	arg.IsMasterRole = flag.Bool("master", false, "")
	arg.IsMinionRole = flag.Bool("minion", false, "")
	flag.Parse()
	if arg.Server == nil || *arg.Server == "" {
		logrus.Fatalf("\"--server\" argument is required for initializing lightning-monkey agent.")
	}
	if arg.MetadataId == nil || *arg.MetadataId == "" {
		logrus.Fatalf("\"--metadata\" argument is required for initializing lightning-monkey agent.")
	}
	if arg.ClusterId == nil || *arg.ClusterId == "" {
		logrus.Fatalf("\"--cluster\" argument is required for initializing lightning-monkey agent.")
	}
	if !*arg.IsETCDRole && !*arg.IsMasterRole && !*arg.IsMinionRole {
		logrus.Fatalf("you must specify one role at least to initialize lightning-monkey agent.")
	}
	if arg.Address == nil {
		ip := GetLocalIP()
		arg.Address = &ip
	}
	agent := LightningMonkeyAgent{}
	agent.Initialize(arg)
	agent.Start()
}

type AgentArgs struct {
	Server       *string
	MetadataId   *string
	ClusterId    *string
	Address      *string
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
