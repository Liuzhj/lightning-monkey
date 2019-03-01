package certs

import (
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
)

func Register(app *iris.Application) error {
	logrus.Infof("    Registering Cluster Certificate Mgmt APIs...")
	app.Get("/apis/v1/certs/get", DownloadCerts)
	return nil
}

func DownloadCerts(ctx iris.Context) {}
