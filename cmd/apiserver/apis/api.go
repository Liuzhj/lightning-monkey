package apis

import (
	v1agent "github.com/g0194776/lightningmonkey/cmd/apiserver/apis/v1/agents"
	v1cert "github.com/g0194776/lightningmonkey/cmd/apiserver/apis/v1/certs"
	v1cluster "github.com/g0194776/lightningmonkey/cmd/apiserver/apis/v1/clusters"
	"github.com/g0194776/lightningmonkey/cmd/apiserver/apis/v1/registry"
	"github.com/kataras/iris"
)

type APIRegisterationManager struct {
	apiEntries []func(app *iris.Application) error
}

func (arm *APIRegisterationManager) Initialize() {
	arm.apiEntries = append(arm.apiEntries, v1cluster.Register)
	arm.apiEntries = append(arm.apiEntries, v1agent.Register)
	arm.apiEntries = append(arm.apiEntries, v1cert.Register)
	arm.apiEntries = append(arm.apiEntries, registry.Register)
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
