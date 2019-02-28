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
	cluster := entities.Cluster{}
	httpData, err := getBody(ctx)
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
	err = managers.NewCluster(&cluster)
	if err != nil {
		rsp = entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp = entities.CreateClusterResponse{
		Response: entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()},
		Cluster:  &cluster,
	}
	_, _ = ctx.JSON(rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
	return
}

func UpdateCluster(ctx iris.Context)    {}
func GetClusterList(ctx iris.Context)   {}
func GetClusterStatus(ctx iris.Context) {}
func getBody(ctx iris.Context) ([]byte, error) {
	return ioutil.ReadAll(ctx.Request().Body)
}
