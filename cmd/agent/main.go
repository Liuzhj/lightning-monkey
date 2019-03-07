package main

import (
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func main() {
	arg := AgentArgs{}
	arg.Server = flag.String("server", "", "api address")
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
