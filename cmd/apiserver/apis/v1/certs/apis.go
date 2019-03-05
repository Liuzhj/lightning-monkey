package certs

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
)

func Register(app *iris.Application) error {
	logrus.Infof("    Registering Cluster Certificate Mgmt APIs...")
	app.Get("/apis/v1/certs/get", DownloadCerts)
	return nil
}

func DownloadCerts(ctx iris.Context) {
	cluster := ctx.URLParam("cluster")
	certName := ctx.URLParam("cert")
	if cluster == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"cluster\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if certName == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"cert\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	//TODO: do more secruity checks.
	certs, err := managers.GetClusterCertificates(cluster)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.InternalError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	var content string
	if content = certs.GetCertificateContent(certName); content == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: fmt.Sprintf("certificate: %s not found.", certName)}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp := entities.GetCertificateResponse{
		Response: entities.Response{
			ErrorId: entities.Succeed,
			Reason:  "",
		},
		Content: content,
	}
	ctx.JSON(&rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}
