package clusters

import (
	"encoding/json"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

func Register(app *iris.Application) error {
	logrus.Infof("    Registering Clusters Mgmt APIs...")
	app.Get("/apis/v1/cluster/list", GetClusterList)
	app.Post("/apis/v1/cluster/create", NewCluster)
	app.Put("/apis/v1/cluster/update", UpdateCluster)
	app.Get("/apis/v1/cluster/status", GetClusterStatus)
	return nil
}

func NewCluster(ctx iris.Context) {
	var rsp interface{}
	cluster := entities.LightningMonkeyClusterSettings{}
	httpData, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	err = json.Unmarshal(httpData, &cluster)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	clusterId, err := managers.NewCluster(&cluster)
	if err != nil {
		rsp = entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp = entities.CreateClusterResponse{
		Response:  entities.Response{ErrorId: entities.Succeed, Reason: ""},
		ClusterId: clusterId,
	}
	_, _ = ctx.JSON(rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func UpdateCluster(ctx iris.Context)    {}
func GetClusterList(ctx iris.Context)   {}
func GetClusterStatus(ctx iris.Context) {}
