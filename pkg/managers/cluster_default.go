package managers

import (
	"github.com/g0194776/lightningmonkey/pkg/entities"
)

var (
	def_processors []func(cluster *entities.LightningMonkeyClusterSettings) error
)

func SetDefaultValue(cluster *entities.LightningMonkeyClusterSettings) error {
	if def_processors == nil || len(def_processors) == 0 {
		return nil
	}
	var err error
	for i := 0; i < len(def_processors); i++ {
		err = def_processors[i](cluster)
		if err != nil {
			return err
		}
	}
	return nil
}
