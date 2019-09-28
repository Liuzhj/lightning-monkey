package main

import (
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

type HelloResponse struct {
}

func (a *LightningMonkeyAgent) InitializeWebServer() {
	app := iris.New()
	app.Get("/hello", HealthCheck)
	app.Post("/system/routes", GenerateSystemRoutingRules)
	logrus.Infof("Starting Web Server...")
	app.Run(iris.Addr(fmt.Sprintf("0.0.0.0:%d", *a.arg.ListenPort)))
}

func GenerateSystemRoutingRules(ctx context.Context) {
	req := entities.GenerateSystemRoutingRulesRequest{}
	httpData, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Next()
		return
	}
	err = json.Unmarshal(httpData, &req)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Next()
		return
	}
	logrus.Infof("Setting system routing rules: %#v", req)
	rsp := entities.Response{ErrorId: 0}
	_, _ = ctx.JSON(rsp)
	ctx.Next()
	return
}

func HealthCheck(ctx iris.Context) {
	_, _ = ctx.JSON(HelloResponse{})
	ctx.Next()
	return
}
