package main

import "github.com/g0194776/lightningmonkey/pkg/entities"

var (
	//CERTIFICATE_STORAGE_PATH = "/etc/kubernetes"
	CERTIFICATE_STORAGE_PATH = "/Users/kevinyang/Documents/certs"
)

type LightningMonkeyAgentReportStatus struct {
	Item entities.LightningMonkeyAgentReportStatusItem
	Key  string
}
