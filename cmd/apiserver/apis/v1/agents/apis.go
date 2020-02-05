package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/cache"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"strconv"
	"sync"
	"time"

	"github.com/g0194776/lightningmonkey/pkg/common"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/kataras/iris"
	"github.com/sirupsen/logrus"
)

type PendingTask struct {
	isCancelled      bool
	cancel           func()
	workProc         func()
	ct               context.Context
	agentId          string
	clusterId        string
	briefInformation entities.LightningMonkeyAgentBriefInformation
}

func (t *PendingTask) Cancel() {
	t.isCancelled = true
	t.cancel()
}

func (t *PendingTask) DoAsync() {
	go func() {
		select {
		case <-t.ct.Done():
			var isOK bool
			var pendingTaskCollection map[string] /*agent id*/ *PendingTask
			//remove itself from the background task list.
			pendingTaskLock.Lock()
			if pendingTaskCollection, isOK = pendingTasks[t.clusterId]; isOK && len(pendingTaskCollection) > 0 {
				if _, isOK = pendingTaskCollection[t.agentId]; isOK {
					delete(pendingTaskCollection, t.agentId)
				}
				//reset
				if len(pendingTaskCollection) > 0 {
					pendingTasks[t.clusterId] = pendingTaskCollection
				} else {
					delete(pendingTasks, t.clusterId)
				}
			}
			pendingTaskLock.Unlock()
			if t.isCancelled {
				return
			}
		}
		t.workProc()
	}()
}

var (
	pendingTaskLock *sync.RWMutex
	pendingTasks    map[string] /*cluster id*/ map[string] /*agent id*/ *PendingTask
)

func Register(app *iris.Application) error {
	if pendingTasks == nil {
		pendingTasks = make(map[string] /*cluster id*/ map[string] /*agent id*/ *PendingTask)
	}
	if pendingTaskLock == nil {
		pendingTaskLock = &sync.RWMutex{}
	}
	logrus.Infof("    Registering Agents Mgmt APIs...")
	app.Post("/apis/v1/agent/register", RegisterAgent)
	app.Get("/apis/v1/agent/query", AgentQueryNextWork)
	app.Put("/apis/v1/agent/status", ReportStatus)
	app.Put("/apis/v1/agent/change", ChangeAgentClusterAndRoles)
	app.Delete("/apis/v1/agent/change", CancelChangeAgentClusterAndRoles)
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
	waitTimeSecs := ctx.URLParamInt32Default("wait", 0)
	if waitTimeSecs > 0 {
		var innerRsp entities.Response
		err := addPendingTask(agentId, waitTimeSecs, newClusterId, oldClusterId, agent, isETCDRole, isMasterRole, isMinionRole, isHARole)
		if err != nil {
			innerRsp = entities.Response{ErrorId: entities.OperationFailed, Reason: err.Error()}
		} else {
			innerRsp = entities.Response{ErrorId: entities.Succeed, Reason: fmt.Sprintf("Waiting, After %d seconds the agent %s will be change to cluster %s!", waitTimeSecs, agentId, newClusterId)}
		}
		ctx.JSON(&innerRsp)
		ctx.Values().Set(entities.RESPONSEINFO, &innerRsp)
		ctx.Next()
		return
	}
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

func CancelChangeAgentClusterAndRoles(ctx iris.Context) {
	agentId := ctx.URLParam("agent-id")
	if agentId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"agent-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	clusterId := ctx.URLParam("cluster-id")
	if clusterId == "" {
		rsp := entities.Response{ErrorId: entities.ParameterError, Reason: "\"cluster-id\" parameter is required."}
		ctx.JSON(&rsp)
		ctx.Values().Set(entities.RESPONSEINFO, &rsp)
		ctx.Next()
		return
	}
	pendingTaskLock.Lock()
	defer pendingTaskLock.Unlock()
	var isOK bool
	var t *PendingTask
	var pendingTaskCollection map[string] /*agent id*/ *PendingTask
	if pendingTaskCollection, isOK = pendingTasks[clusterId]; isOK && len(pendingTaskCollection) > 0 {
		if t, isOK = pendingTaskCollection[agentId]; isOK {
			delete(pendingTaskCollection, agentId)
			t.Cancel()
		}
		//reset
		if len(pendingTaskCollection) > 0 {
			pendingTasks[clusterId] = pendingTaskCollection
		} else {
			delete(pendingTasks, clusterId)
		}
	}
	rsp := entities.Response{ErrorId: entities.Succeed}
	ctx.JSON(&rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func addPendingTask(agentId string, waitTimeSecs int32, newClusterId, oldClusterId string, agent *entities.LightningMonkeyAgent, isETCDRole, isMasterRole, isMinionRole, isHARole bool) error {
	pendingTaskLock.Lock()
	defer pendingTaskLock.Unlock()

	var isOK bool
	var t *PendingTask
	var pendingTaskCollection map[string] /*agent id*/ *PendingTask
	if pendingTaskCollection, isOK = pendingTasks[newClusterId]; isOK && len(pendingTaskCollection) > 0 {
		if t, isOK = pendingTaskCollection[agentId]; isOK {
			return fmt.Errorf("Duplicated background task for agent %s!", agentId)
		}
	}
	t = NewPendingTask(agentId, newClusterId, int(waitTimeSecs), func() {
		logrus.Warnf("Delay triggered by background task, changing agent %s to cluster %s...", agentId, newClusterId)
		err := common.ClusterManager.TransferAgentToCluster(
			oldClusterId,
			newClusterId,
			agent,
			isETCDRole,
			isMasterRole,
			isMinionRole,
			isHARole)
		if err != nil {
			if agent.State != nil {
				msg := fmt.Sprintf("Failed to change agent %s to cluster %s, error: %s", agentId, newClusterId, err.Error())
				logrus.Error(msg)
				agent.State.Reason = msg
			}
		}
	})
	if pendingTaskCollection == nil {
		pendingTaskCollection = make(map[string] /*agent id*/ *PendingTask)
	}
	t.briefInformation = entities.LightningMonkeyAgentBriefInformation{
		Id:              agent.Id,
		HasETCDRole:     isETCDRole,
		HasMasterRole:   isMasterRole,
		HasMinionRole:   isMinionRole,
		HasHARole:       isHARole,
		Hostname:        agent.Hostname,
		HostInformation: agent.HostInformation,
		DeploymentPhase: entities.AgentDeploymentPhase_Pending,
		State:           agent.State,
	}
	pendingTaskCollection[agentId] = t
	pendingTasks[newClusterId] = pendingTaskCollection
	t.DoAsync()
	return nil
}

func NewPendingTask(agentId, clusterId string, waitTime int, workProc func()) *PendingTask {
	d, _ := time.ParseDuration(fmt.Sprintf("%ds", waitTime))
	ct, cf := context.WithTimeout(context.Background(), d)
	return &PendingTask{agentId: agentId, clusterId: clusterId, ct: ct, cancel: cf, workProc: workProc}
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
	if agents != nil && len(agents) > 0 {
		pendingTaskLock.RLock()
		if clusterId == uuid.Nil.String() {
			agents = filterPoolingHosts(agents)
		} else {
			agents = filterClusterHosts(cluster, agents)
		}
		pendingTaskLock.RUnlock()
	}
	rsp := entities.GetAgentListResponse{
		Response: entities.Response{ErrorId: entities.Succeed},
		Agents:   agents,
	}
	ctx.JSON(&rsp)
	ctx.Values().Set(entities.RESPONSEINFO, &rsp)
	ctx.Next()
}

func filterPoolingHosts(agents []entities.LightningMonkeyAgentBriefInformation) []entities.LightningMonkeyAgentBriefInformation {
	if pendingTasks == nil || len(pendingTasks) == 0 {
		return agents
	}
	if agents == nil || len(agents) == 0 {
		return agents
	}
	var hasFound bool
	var pendingTaskCollection map[string] /*agent id*/ *PendingTask
	var filterAgents []entities.LightningMonkeyAgentBriefInformation
	for i := 0; i < len(agents); i++ {
		for _, pendingTaskCollection = range pendingTasks {
			if _, hasFound = pendingTaskCollection[agents[i].Id]; hasFound {
				break
			}
		}
		if !hasFound {
			filterAgents = append(filterAgents, agents[i])
		}
	}
	return filterAgents
}

func filterClusterHosts(cluster cache.ClusterController, agents []entities.LightningMonkeyAgentBriefInformation) []entities.LightningMonkeyAgentBriefInformation {
	var isOK bool
	var pendingTaskCollection map[string] /*agent id*/ *PendingTask
	if pendingTaskCollection, isOK = pendingTasks[cluster.GetClusterId()]; !isOK {
		return agents
	}
	for _, t := range pendingTaskCollection {
		agents = append(agents, t.briefInformation)
	}
	return agents
}
