package apis

import (
	v1cluster "github.com/g0194776/lightningmonkey/cmd/apiserver/apis/v1/clusters"
	"github.com/kataras/iris"
)

type APIRegisterationManager struct {
	apiEntries []func(app *iris.Application) error
}

func (arm *APIRegisterationManager) Initialize() {
	arm.apiEntries = append(arm.apiEntries, v1cluster.Register)
}

func (arm *APIRegisterationManager) DoRegister(app *iris.Application) error {
	var err error
	for _, registerFunc := range arm.apiEntries {
		err = registerFunc(app)
		if err != nil {
			return err
		}
	}
	return nil
}
