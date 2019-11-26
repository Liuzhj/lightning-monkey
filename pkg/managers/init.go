package managers

import (
	"errors"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
)

func init() {
	setDefaultValueProcessors()
	setBizCheckProcessor()
	logrus.Info("Built-in processors loaded!")
}

func setDefaultValueProcessors() {
	//set default node port range.
	def_processors = append(def_processors, func(cluster *entities.LightningMonkeyClusterSettings) error {
		if cluster.PortRangeSettings == nil {
			cluster.PortRangeSettings = &entities.NodePortRangeSettings{
				Begin: 30000,
				End:   32767,
			}
		}
		return nil
	})
}

func setBizCheckProcessor() {
	//HA settings check.
	check_processors = append(check_processors, func(cluster entities.LightningMonkeyClusterSettings) error {
		if cluster.HASettings != nil {
			if cluster.HASettings.VIP == "" {
				return errors.New("\"ha_settings.vip\" is needed for initializing HAProxy & KeepAlived installation.")
			}
			if cluster.HASettings.NodeCount <= 0 {
				return errors.New("\"ha_settings.count\" must greater than zero!")
			}
		}
		return nil
	})
	//port range check.
	check_processors = append(check_processors, func(cluster entities.LightningMonkeyClusterSettings) error {
		if cluster.PortRangeSettings.Begin == 0 {
			return errors.New("Illegal node port range, \"node_port_range_settings.begin\" must greater than zero!")
		}
		if cluster.PortRangeSettings.End == 0 {
			return errors.New("Illegal node port range, \"node_port_range_settings.end\" must greater than zero!")
		}
		if cluster.PortRangeSettings.End <= cluster.PortRangeSettings.Begin {
			return errors.New("Illegal node port range, \"node_port_range_settings.end\" must greater than \"node_port_range_settings.begin\"!")
		}
		return nil
	})
	//image pulling secrets check.
	check_processors = append(check_processors, func(cluster entities.LightningMonkeyClusterSettings) error {
		if cluster.ImagePullSecrets != nil && len(cluster.ImagePullSecrets) > 0 {
			for i := 0; i < len(cluster.ImagePullSecrets); i++ {
				ps := cluster.ImagePullSecrets[i]
				if ps.Name == "" || ps.Username == "" || ps.Password == "" || ps.Email == "" || ps.Namespace == "" || ps.Registry == "" {
					return errors.New("All of fields value in each of image pulling secrets are required!")
				}
			}
		}
		return nil
	})
}
