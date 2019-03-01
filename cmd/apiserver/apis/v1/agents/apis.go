package agents

import (
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
)

func Register(app *iris.Application) error {
	logrus.Infof("    Registering Agents Mgmt APIs...")
	app.Post("/apis/v1/agent/register", RegisterAgent)
	return nil
}

func RegisterAgent(ctx iris.Context) {}
