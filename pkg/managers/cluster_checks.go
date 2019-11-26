package managers

import "github.com/g0194776/lightningmonkey/pkg/entities"

var (
	check_processors []func(cluster entities.LightningMonkeyClusterSettings) error
)

func SecurityCheck(cluster entities.LightningMonkeyClusterSettings) error {
	if check_processors == nil || len(check_processors) == 0 {
		return nil
	}
	var err error
	for i := 0; i < len(check_processors); i++ {
		err = check_processors[i](cluster)
		if err != nil {
			return err
		}
	}
	return nil
}
