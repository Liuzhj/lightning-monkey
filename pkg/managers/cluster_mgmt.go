package managers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"
)

func NewCluster(cluster *entities.LightningMonkeyClusterSettings) (string, error) {
	var err error
	var certsResources *certs.GeneratedCertsMap
	//not pooling resource.
	if cluster.Id != uuid.Nil.String() {
		//security checks.
		if cluster.ExpectedETCDCount <= 0 {
			return "", errors.New("Expected ETCD node count must greater than 0")
		}
		if cluster.KubernetesVersion == "" {
			return "", errors.New("You must specify expected Kubernetes version!")
		}
		if cluster.ServiceDNSClusterIP == "" {
			return "", errors.New("Field: \"service_dns_cluster_ip\" is required to configure in-cluster DNS communication!")
		}
		_, _, err = net.ParseCIDR(cluster.PodNetworkCIDR)
		if err != nil {
			return "", fmt.Errorf("Failed to parse \"cluster.PodNetworkCIDR\" value as correct CIDR format, error: %s", err.Error())
		}
		_, _, err = net.ParseCIDR(cluster.ServiceCIDR)
		if err != nil {
			return "", fmt.Errorf("Failed to parse \"cluster.ServiceCIDR\" value as correct CIDR format, error: %s", err.Error())
		}
		//generate required certificates.
		certsResources, err = common.CertManager.GenerateMainCACertificates()
		if err != nil {
			return "", fmt.Errorf("Failed to generate Kubernetes required certificates, error: %s", err.Error())
		}
	}
	//considered troubleshooting, set to empty is not an required condition.
	if cluster.Id == "" {
		//reset cluster fields.
		cluster.Id = uuid.NewV4().String()
	}
	if cluster.MaximumAllowedPodCountPerNode <= 0 {
		cluster.MaximumAllowedPodCountPerNode = 110
	}
	if cluster.ServiceDNSDomain == "" {
		cluster.ServiceDNSDomain = "cluster.local"
	}
	cluster.SecurityToken = "abc"
	cluster.CreateTime = time.Now()
	err = saveCluster(*cluster, certsResources)
	if err != nil {
		return "", fmt.Errorf("Failed to save cluster information to storage driver, error: %s", err.Error())
	}
	return cluster.Id, err
}

func GetClusterCertificates(clusterId string) (entities.LightningMonkeyCertificateCollection, error) {
	return common.ClusterManager.GetClusterCertificates(clusterId)
}

func saveCluster(cluster entities.LightningMonkeyClusterSettings, certsMap *certs.GeneratedCertsMap) error {
	//STEP 1, add generated cluster certificates.
	err := saveClusterCertificate(cluster, certsMap)
	if err != nil {
		return err
	}

	//STEP 2, create cluster metadata
	//after writing certificates to add metadata is used for avoiding cache missing.
	//that's very important to ensure that all newest events can be received successfully from ETCD watcher.
	path := fmt.Sprintf("/lightning-monkey/clusters/%s/metadata", cluster.Id)
	data, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), common.StorageDriver.GetRequestTimeoutDuration())
	defer cancel()
	_, err = common.StorageDriver.Put(ctx, path, string(data))
	return err
}

func saveClusterCertificate(cluster entities.LightningMonkeyClusterSettings, certsMap *certs.GeneratedCertsMap) error {
	var path string
	var err error
	if certsMap == nil || certsMap.GetResources() == nil || len(certsMap.GetResources()) == 0 {
		logrus.Warnf("No any generated certificates should save to remote storage, cluster: %s", cluster.Id)
		return nil
	}
	cm := certsMap.GetResources()
	for k, v := range cm {
		path = fmt.Sprintf("/lightning-monkey/clusters/%s/certificates/%s", cluster.Id, strings.Replace(k, "/", "_", -1))
		_, err = common.StorageDriver.Put(context.Background(), path, v)
		if err != nil {
			return err
		}
	}
	return nil
}
