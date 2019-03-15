package agents

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
	logrus.Infof("    Registering Agents Mgmt APIs...")
	app.Post("/apis/v1/agent/register", RegisterAgent)
	app.Get("/apis/v1/agent/query", AgentQueryNextWork)
	app.Put("/apis/v1/agent/status", ReportStatus)
	return nil
}

func RegisterAgent(ctx iris.Context) {
	var rsp interface{}
	agent := entities.Agent{}
	httpData, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	err = json.Unmarshal(httpData, &agent)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if agent.LastReportIP == "" {
		agent.LastReportIP = ctx.RemoteAddr()
	}
	agent.LastReportStatus = entities.AgentStatus_Provisioning
	cluster, err := managers.RegisterAgent(&agent)
	if err != nil {
		rsp = entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp = entities.RegisterAgentResponse{
		Response:    entities.Response{ErrorId: entities.Succeed, Reason: ""},
		BasicImages: common.BasicImages["1.12.5"], /*test only*/
		MasterSettings: map[string]string{
			entities.MasterSettings_PodCIDR:           cluster.PodNetworkCIDR,
			entities.MasterSettings_ServiceCIDR:       cluster.ServiceCIDR,
			entities.MasterSettings_ServiceDNSDomain:  cluster.ServiceDNSDomain,
			entities.MasterSettings_KubernetesVersion: cluster.KubernetesVersion,
		},
	}
	_, _ = ctx.JSON(rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func AgentQueryNextWork(ctx iris.Context) {
	metadataId := ctx.URLParam("metadata-id")
	if metadataId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"metadata-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	job, err := managers.QueryAgentNextWorkItem(metadataId)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp := entities.GetNextAgentJobResponse{Response: entities.Response{ErrorId: entities.Succeed, Reason: ""}, Job: job}
	_, _ = ctx.JSON(rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func ReportStatus(ctx iris.Context) {
	metadataId := ctx.URLParam("metadata-id")
	if metadataId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"metadata-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	status := entities.AgentStatus{}
	httpData, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	err = json.Unmarshal(httpData, &status)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if status.IP == "" {
		status.IP = ctx.RemoteAddr()
	}
	logrus.Infof("[Report]: IP: %s, Type: %s, Status: [%s], Reason: %s", status.IP, status.ReportType, status.Status, status.Reason)
	err = managers.AgentReportStatus(metadataId, status)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp := entities.Response{ErrorId: entities.Succeed}
	ctx.JSON(&rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}
