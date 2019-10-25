package agents

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
)

func Register(app *iris.Application) error {
	logrus.Infof("    Registering Agents Mgmt APIs...")
	app.Post("/apis/v1/agent/register", RegisterAgent)
	app.Get("/apis/v1/agent/query", AgentQueryNextWork)
	app.Put("/apis/v1/agent/status", ReportStatus)
	app.Put("/apis/v1/agent/change", ChangeAgentClusterAndRoles)
	app.Get("/apis/v1/agents/list", ListAgentsByClusterId)
	return nil
}

func RegisterAgent(ctx iris.Context) {
	var rsp interface{}
	agent := entities.LightningMonkeyAgent{}
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
	agent.State = &entities.AgentState{}
	agent.State.LastReportIP = ctx.RemoteAddr()
	settings, agentId, clusterId, leaseId, err := managers.RegisterAgent(&agent)
	if err != nil {
		rsp = entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	vip := ""
	if settings.HASettings != nil {
		vip = settings.HASettings.VIP
	}
	r := entities.RegisterAgentResponse{
		Response:    entities.Response{ErrorId: entities.Succeed, Reason: ""},
		AgentId:     agentId,
		ClusterId:   clusterId,
		LeaseId:     leaseId,
		BasicImages: common.BasicImages[settings.KubernetesVersion],
		MasterSettings: map[string]string{
			entities.MasterSettings_PodCIDR:               settings.PodNetworkCIDR,
			entities.MasterSettings_ServiceCIDR:           settings.ServiceCIDR,
			entities.MasterSettings_ServiceDNSDomain:      settings.ServiceDNSDomain,
			entities.MasterSettings_ServiceDNSClusterIP:   settings.ServiceDNSClusterIP,
			entities.MasterSettings_KubernetesVersion:     settings.KubernetesVersion,
			entities.MasterSettings_APIServerVIP:          vip,
			entities.MasterSettings_MaxPodCountPerNode:    strconv.Itoa(settings.MaximumAllowedPodCountPerNode),
			entities.MasterSettings_ExpectedETCDNodeCount: strconv.Itoa(settings.ExpectedETCDCount),
			entities.MasterSettings_DockerRegistry:        "mirrorgooglecontainers/hyperkube",
			entities.MasterSettings_DockerExtraGraphPath:  settings.ExtraDockerGraphPath,
			entities.MasterSettings_PortRange:             fmt.Sprintf("%d-%d", settings.PortRangeSettings.Begin, settings.PortRangeSettings.End),
		},
	}
	if settings.ResourceReservation != nil {
		r.MasterSettings[entities.MasterSettings_ResourceReservation_Kube] = settings.ResourceReservation.Kube
		r.MasterSettings[entities.MasterSettings_ResourceReservation_System] = settings.ResourceReservation.System
	}
	r.BasicImages.HTTPDownloadToken = entities.HTTPDockerImageDownloadToken
	rsp = r
	_, _ = ctx.JSON(rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func AgentQueryNextWork(ctx iris.Context) {
	clusterId := ctx.URLParam("cluster-id")
	if clusterId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"cluster-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	agentId := ctx.URLParam("agent-id")
	if agentId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"agent-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}

	job, err := managers.QueryAgentNextWorkItem(clusterId, agentId)
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
	agentId := ctx.URLParam("agent-id")
	if agentId == "" {
		rsp := entities.AgentReportStatusResponse{Response: entities.Response{ErrorId: entities.ParameterError, Reason: "\"agent-id\" parameter is required."}}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	clusterId := ctx.URLParam("cluster-id")
	if clusterId == "" {
		rsp := entities.AgentReportStatusResponse{Response: entities.Response{ErrorId: entities.ParameterError, Reason: "\"cluster-id\" parameter is required."}}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	status := entities.LightningMonkeyAgentReportStatus{}
	httpData, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		rsp := entities.AgentReportStatusResponse{Response: entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	err = json.Unmarshal(httpData, &status)
	if err != nil {
		rsp := entities.AgentReportStatusResponse{Response: entities.Response{ErrorId: entities.DeserializeError, Reason: err.Error()}}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if status.IP == "" {
		status.IP = ctx.RemoteAddr()
	}
	logrus.Debugf("[Report]: IP: %s, Status: %#v", status.IP, status.Items)
	leaseId, err := managers.AgentReportStatus(clusterId, agentId, status)
	if err != nil {
		rsp := entities.AgentReportStatusResponse{Response: entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp := entities.AgentReportStatusResponse{Response: entities.Response{ErrorId: entities.Succeed}, LeaseId: leaseId}
	ctx.JSON(&rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func ChangeAgentClusterAndRoles(ctx iris.Context) {
	agentId := ctx.URLParam("agent-id")
	if agentId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"agent-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	oldClusterId := ctx.URLParam("old-cluster-id")
	if oldClusterId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"old-cluster-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	newClusterId := ctx.URLParam("new-cluster-id")
	if newClusterId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"new-cluster-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	cluster, err := common.ClusterManager.GetClusterById(oldClusterId)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.InternalError, Reason: fmt.Sprintf("Failed to retrieve cluster information from cache, error: %s", err.Error())}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if cluster == nil {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: fmt.Sprintf("Cluster: %s not found!", oldClusterId)}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	agent, err := cluster.GetCachedAgent(agentId)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.InternalError, Reason: fmt.Sprintf("Failed to retrieve agent information from cache, cluster-id: %s, agent-id: %s, error: %s", oldClusterId, agentId, err.Error())}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if agent == nil {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: fmt.Sprintf("Agent: %s not found!", agentId)}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	isETCDRole := ctx.URLParamInt32Default("etcd", 0) == 1
	isMasterRole := ctx.URLParamInt32Default("master", 0) == 1
	isMinionRole := ctx.URLParamInt32Default("minion", 0) == 1
	isHARole := ctx.URLParamInt32Default("ha", 0) == 1
	err = common.ClusterManager.TransferAgentToCluster(
		oldClusterId,
		newClusterId,
		agent,
		isETCDRole,
		isMasterRole,
		isMinionRole,
		isHARole)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.InternalError, Reason: err.Error()}
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

func ListAgentsByClusterId(ctx iris.Context) {
	clusterId := ctx.URLParam("cluster-id")
	if clusterId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"cluster-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	cluster, err := common.ClusterManager.GetClusterById(clusterId)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.InternalError, Reason: fmt.Sprintf("Failed to retrieve cluster(%s) information from cache, error: %s", clusterId, err.Error())}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	if cluster == nil {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: fmt.Sprintf("Cluster: %s not found!", clusterId)}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	agents, err := cluster.GetAgentList(false)
	if err != nil {
		rsp := entities.Response{ErrorId: entities.InternalError, Reason: fmt.Sprintf("Failed to list agents from cluster(%s), error: %s", clusterId, err.Error())}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	rsp := entities.GetAgentListResponse{
		Response: entities.Response{ErrorId: entities.Succeed},
		Agents:   agents,
	}
	ctx.JSON(&rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}
