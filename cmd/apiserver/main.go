package main

import (
	"github.com/g0194776/lightningmonkey/cmd/apiserver/apis"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

func main() {
	logrus.Infof("Lightning Monkey(v1.0.0)")
	logrus.Infof("Registering APIs...")
	app := iris.New()
	arm := apis.APIRegisterationManager{}
	arm.Initialize()
	err := arm.DoRegister(app)
	if err != nil {
		logrus.Fatalf("Failed to register API group to web engine, error: %s", err.Error())
		return
	}
	logrus.Infof("Creating backend storage driver...")
	driverType := os.Getenv("BACKEND_STORAGE_TYPE")
	if driverType == "" {
		driverType = "mongo"
	}
	sdf := storage.StorageDriverFactory{}
	//handle args.
	driverArgsStr := os.Getenv("BACKEND_STORAGE_ARGS")
	driverArgs := map[string]string{}
	if driverArgsStr != "" {
		arr := strings.Split(driverArgsStr, ";")
		if arr != nil && len(arr) > 0 {
			for i := 0; i < len(arr); i++ {
				pairs := strings.Split(arr[i], "=")
				driverArgs[pairs[0]] = pairs[1]
			}
		}
	}
	driver, err := sdf.NewStorageDriver(driverType)
	if err != nil {
		logrus.Fatalf("Failed to create backend storage driver with: %s, error: %s", driverType, err.Error())
		return
	}
	err = driver.Initialize(driverArgs)
	if err != nil {
		logrus.Fatalf("Failed to initialize backend storage driver: %s, error: %s", driverType, err.Error())
		return
	}
	logrus.Infof("Starting Web Engine...")
	go app.Run(iris.Addr(":8080"))
	select {}
}
