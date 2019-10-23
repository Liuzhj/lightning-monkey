package clusters

import (
	"encoding/json"
	"github.com/g0194776/lightningmonkey/pkg/common"
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
	app.Get("/apis/v1/cluster/status", GetClusterComponentStatus)
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
	//HA settings check.
	if cluster.HASettings != nil {
		if cluster.HASettings.VIP == "" {
			rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"VIP\" is needed for initializing HAProxy & KeepAlived installation."}
			ctx.JSON(&rsp)
			ctx.Values().Set(entities.RESPONSEINFO, &rsp)
			ctx.Next()
			return
		}
		if cluster.HASettings.NodeCount <= 0 {
			rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"Count\" must greater than zero!"}
			ctx.JSON(&rsp)
			ctx.Values().Set(entities.RESPONSEINFO, &rsp)
			ctx.Next()
			return
		}
	}
	//node port range check.
	if cluster.PortRangeSettings == nil {
		cluster.PortRangeSettings = &entities.NodePortRangeSettings{
			Begin: 30000,
			End:   32767,
		}
	}
	if cluster.PortRangeSettings.Begin == 0 {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "Illegal node port range, \"node_port_range_settings.begin\" must greater than zero!"}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if cluster.PortRangeSettings.End == 0 {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "Illegal node port range, \"node_port_range_settings.end\" must greater than zero!"}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if cluster.PortRangeSettings.End <= cluster.PortRangeSettings.Begin {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "Illegal node port range, \"node_port_range_settings.end\" must greater than \"node_port_range_settings.begin\"!"}
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

func GetClusterComponentStatus(ctx iris.Context) {
	clusterId := ctx.URLParam("cluster-id")
	if clusterId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"cluster-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	c, err := common.ClusterManager.GetClusterById(clusterId)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.InternalError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	wps := c.GetWachPoints()
	rsp := entities.GetClusterComponentStatusResponse{
		Response:    entities.Response{ErrorId: entities.Succeed, Reason: ""},
		WatchPoints: wps,
	}
	_, _ = ctx.JSON(rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func UpdateCluster(ctx iris.Context)  {}
func GetClusterList(ctx iris.Context) {}
