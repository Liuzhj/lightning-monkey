package main

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/cmd/apiserver/apis"
	"github.com/g0194776/lightningmonkey/pkg/cache"
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/g0194776/lightningmonkey/pkg/storage"
	"github.com/kataras/iris"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"math/rand"
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
	entities.HTTPDockerImageDownloadToken = os.Getenv("GET_TOKEN")
	logrus.Infof("Creating backend storage driver...")
	driverType := os.Getenv("BACKEND_STORAGE_TYPE")
	if driverType == "" {
		driverType = "etcd"
	}
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		l, err := logrus.ParseLevel(logLevel)
		if err != nil {
			logrus.Fatalf("Unable to set log level to logrus: %s, error: %s", logLevel, err.Error())
			return
		}
		logrus.SetLevel(l)
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
	common.StorageDriver = driver
	logrus.Infof("Initializing cluster manager...")
	common.ClusterManager = &cache.ClusterManager{}
	err = common.ClusterManager.Initialize(driver)
	if err != nil {
		logrus.Fatalf("Failed to initialize cluster manager, error: %s", err.Error())
		return
	}
	common.CertManager = &certs.CertificateManagerImple{}
	//generates readonly token for downloading payloads.
	if entities.HTTPDockerImageDownloadToken == "" {
		count := 24
		b := make([]byte, count)
		if _, err := rand.Read(b); err != nil {
			logrus.Fatalf("Could not generate token, error: %s", err.Error())
			return
		}
		entities.HTTPDockerImageDownloadToken = fmt.Sprintf("%x", b)
		logrus.Info("*** Please kindly record this auto-generated token for downloading the deployment payloads ***")
		logrus.Info("***")
		logrus.Info("*** " + entities.HTTPDockerImageDownloadToken)
		logrus.Info("***")
	} else {
		logrus.Infof("*** Please kindly record this user-specified token for downloading the deployment payloads ***")
		logrus.Info("***")
		logrus.Info("*** " + entities.HTTPDockerImageDownloadToken)
		logrus.Info("***")
	}
	logrus.Infof("Creating resource pool...")
	_, err = managers.NewCluster(&entities.LightningMonkeyClusterSettings{Id: uuid.Nil.String()})
	if err != nil {
		logrus.Fatalf("Failed to create resource pool, error: %s", err.Error())
		return
	}
	logrus.Infof("Starting Web Engine...")
	app.Run(iris.Addr("0.0.0.0:8080"))
}
