package managers

import (
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/globalsign/mgo/bson"
	"net"
	"time"
)

func NewCluster(cluster *entities.Cluster) error {
	//security checks.
	if cluster.ExpectedETCDCount <= 0 {
		return errors.New("Expected ETCD node count must greater than 0")
	}
	if cluster.KubernetesVersion == "" {
		return errors.New("You must specify expected Kubernetes version!")
	}
	if cluster.ServiceDNSDomain == "" /* default: "cluster.local" */ {
		return errors.New("You must specify cluster DNS domain!")
	}
	_, _, err := net.ParseCIDR(cluster.PodNetworkCIDR)
	if err != nil {
		return fmt.Errorf("Failed to parse \"cluster.PodNetworkCIDR\" value as correct CIDR format, error: %s", err.Error())
	}
	_, _, err = net.ParseCIDR(cluster.ServiceCIDR)
	if err != nil {
		return fmt.Errorf("Failed to parse \"cluster.ServiceCIDR\" value as correct CIDR format, error: %s", err.Error())
	}
	//generate required certificates.
	certsResources, err := certs.GenerateMainCACertificates()
	if err != nil {
		return fmt.Errorf("Failed to generate Kubernetes required certificates, error: %s", err.Error())
	}
	//reset cluster fields.
	clusterId := bson.NewObjectId()
	cluster.Status = "NEW"
	cluster.Id = &clusterId
	cluster.SecurityToken = "abc"
	cluster.CreateTime = time.Now()
	err = common.StorageDriver.SaveCluster(cluster, certsResources)
	if err != nil {
		return fmt.Errorf("Failed to save cluster information to storage driver, error: %s", err.Error())
	}
	return err
}

func GetClusterCertificates(clusterId string) (entities.CertificateCollection, error) {
	return common.StorageDriver.GetCertificatesByClusterId(clusterId)
}
