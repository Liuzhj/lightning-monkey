package managers

import (
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/entities"
)

func NewCluster(cluster *entities.Cluster) error {
	_, err := certs.GenerateMasterCertificates("/tmp/sdf", "0.0.0.0", "192.168.1.1/12")
	return err
}
