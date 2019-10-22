package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/matishsiao/goInfo"
	uuid "github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/g0194776/lightningmonkey/pkg/managers"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"k8s.io/apimachinery/pkg/util/json"
)

var (
	errNotInitialized error = errors.New("Not specified any cluster-id or roles.")
)

func (a *LightningMonkeyAgent) Register() (err error) {
	if atomic.LoadInt32(&a.hasRegistered) == 1 {
		return nil
	}
	if atomic.LoadInt32(&a.hasRegistered) == -1 {
		return errNotInitialized
	}
	defer func() {
		if err == nil {
			if atomic.CompareAndSwapInt32(&a.hasRegistered, 0, 1) {
				a.lastRegisteredTime = time.Now()
			}
		} else {
			if err == errNotInitialized {
				atomic.SwapInt32(&a.hasRegistered, -1)
			} else {
				atomic.SwapInt32(&a.hasRegistered, 0)
			}
		}
	}()
	hostname, _ := os.Hostname()
	agentObj := entities.LightningMonkeyAgent{
		HasETCDRole:   *a.arg.IsETCDRole,
		HasMasterRole: *a.arg.IsMasterRole,
		HasMinionRole: *a.arg.IsMinionRole,
		HasHARole:     *a.arg.IsHARole,
		ClusterId:     *a.arg.ClusterId,
		Hostname:      hostname,
		ListenPort:    *a.arg.ListenPort,
		Id:            a.arg.AgentId,
	}
	//obtains host information.
	ci, err := cpu.InfoWithContext(context.Background())
	if err != nil {
		logrus.Fatalf("Failed to obtain host CPU information, error: %s", err.Error())
		return
	}
	memory, err := mem.VirtualMemory()
	if err != nil {
		logrus.Fatalf("Failed to obtain host Memory information, error: %s", err.Error())
		return
	}
	gi := goInfo.GetInfo()
	agentObj.HostInformation.CPUCores = ci[0].Cores
	agentObj.HostInformation.CPUMhz = ci[0].Mhz
	agentObj.HostInformation.MemoryTotalMB = memory.Total / 1024 / 1024
	agentObj.HostInformation.OS = gi.GoOS
	agentObj.HostInformation.Kernel = fmt.Sprintf("%s %s", gi.Kernel, gi.Core)
	bodyData, err := json.Marshal(agentObj)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	client := http.Client{
		Timeout:   time.Second * 120,
		Transport: http.DefaultTransport,
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apis/v1/agent/register", *a.arg.Server), bytes.NewReader(bodyData))
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	rsp, err := client.Do(req)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return xerrors.Errorf("Remote API server returned a non-zero HTTP status code: %d %w", rsp.StatusCode, crashError)
	}
	httpRspBodyDate, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	rspObj := entities.RegisterAgentResponse{}
	err = json.Unmarshal(httpRspBodyDate, &rspObj)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	if rspObj.NeedCrash {
		return xerrors.Errorf("Remote API server returned an unrecoverable error: %s %w", rspObj.Reason, crashError)
	}
	if rspObj.ErrorId != entities.Succeed {
		return fmt.Errorf("Remote API server returned a non-zero biz code: %d %w", rspObj.ErrorId, crashError)
	}
	a.arg.AgentId = rspObj.AgentId
	a.arg.LeaseId = rspObj.LeaseId
	if *a.arg.ClusterId == uuid.Nil.String() || !a.hasInitializedRoles() {
		logrus.Warn("Currently, agent has not belong to any cluster or has no any initialized roles, it's waiting for the remote call...")
		return errNotInitialized
	}
	a.basicImages = &rspObj.BasicImages
	a.masterSettings = rspObj.MasterSettings
	a.dockerImageManager, err = managers.NewDockerImageManager(*a.arg.Server, a.dockerClient, &rspObj.BasicImages)
	if err != nil {
		return xerrors.Errorf("Failed to create new docker image manager: %s %w", err.Error(), crashError)
	}
	//support lazy updating cluster-id which it belongs to.
	if *a.arg.ClusterId == uuid.Nil.String() && rspObj.ClusterId != uuid.Nil.String() {
		a.arg.ClusterId = &rspObj.ClusterId
		if a.rr != nil {
			a.rr.ClusterID = rspObj.ClusterId
			err = a.saveRecoveryFile()
			if err != nil {
				logrus.Fatalf("Failed to save recovery file during updating its cluster-id, error: %s", err.Error())
				return
			}
		}
	}
	a.expectedETCDNodeCount, err = strconv.Atoi(rspObj.MasterSettings[entities.MasterSettings_ExpectedETCDNodeCount])
	if err != nil {
		logrus.Fatal("Illegal number of expected ETCD count: %s", rspObj.MasterSettings[entities.MasterSettings_ExpectedETCDNodeCount])
		return
	}
	logrus.Debugf("API file server readonly token: %s", rspObj.BasicImages.HTTPDownloadToken)
	entities.HTTPDockerImageDownloadToken = rspObj.BasicImages.HTTPDownloadToken
	logrus.Info("Preparing downloading certificates & loading docker images...")
	err = a.downloadCertificates()
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	err = a.dockerImageManager.Ready()
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	//directly start kubelet up when it has not Minion role.
	if !*a.arg.IsMinionRole {
		return a.runKubeletContainer(*a.arg.Address)
	}
	//otherwise, wait until all of depended components has been started.
	return nil
}

func (a *LightningMonkeyAgent) downloadCertificates() error {
	err := os.MkdirAll(CERTIFICATE_STORAGE_PATH, 0755) //"rwxr-xr-x"
	if err != nil {
		return xerrors.Errorf("Failed to create certificate storage path: %s %w", err.Error(), crashError)
	}
	//TODO: does not all of base certificates are needed during minion node initialization.
	neededCerts := []string{
		"ca.crt",
		"ca.key",
		"front-proxy-ca.crt",
		"front-proxy-ca.key",
		"sa.pub",
		"sa.key",
		"etcd/ca.crt",
		"etcd/ca.key",
	}
	for i := 0; i < len(neededCerts); i++ {
		logrus.Infof("Downloading certificate: \"%s\"...", neededCerts[i])
		err = a.saveRemoteCertificate(neededCerts[i], CERTIFICATE_STORAGE_PATH)
		if err != nil {
			return xerrors.Errorf("Failed to save remote certificate data to local disk file, error: %s %w", err.Error(), crashError)
		}
	}
	return nil
}

func (a *LightningMonkeyAgent) saveRemoteCertificate(certName, path string) error {
	client := http.Client{
		Timeout:   time.Second * 5,
		Transport: http.DefaultTransport,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apis/v1/certs/get?cluster=%s&cert=%s", *a.arg.Server, *a.arg.ClusterId, certName), nil)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	rsp, err := client.Do(req)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return xerrors.Errorf("Remote API server returned a non-zero HTTP status code: %d %w", rsp.StatusCode, crashError)
	}
	rspObj := entities.GetCertificateResponse{}
	rspData, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	err = json.Unmarshal(rspData, &rspObj)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	if rspObj.ErrorId != entities.Succeed {
		return fmt.Errorf("Remote API server returned a non-zero biz code: %d %w", rspObj.ErrorId, crashError)
	}
	if rspObj.Content == "" {
		return fmt.Errorf("Empty certificate data: %s, %w", certName, crashError)
	}
	filePath := filepath.Join(path, certName)
	if _, err = os.Stat(filePath); os.IsExist(err) {
		if !rspObj.ForceUpdate {
			return nil
		}
		//delete existed file, re-generate it.
		_ = os.RemoveAll(filePath)
	}
	destPath := filepath.Dir(filePath)
	_ = os.MkdirAll(destPath, 0755)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return fmt.Errorf("Failed to create certificate file: %s, %s, %w", certName, err.Error(), crashError)
	}
	defer f.Close()
	_, err = f.WriteString(rspObj.Content)
	if err != nil {
		return fmt.Errorf("Failed to write certificate data to local disk file: %s, %s, %w", certName, err.Error(), crashError)
	}
	return nil
}

func (a *LightningMonkeyAgent) Initialize(arg AgentArgs) {
	a.arg = &arg
	if a.c == nil {
		a.c = make(chan LightningMonkeyAgentReportStatus)
	}
	if a.ItemsStatus == nil {
		a.ItemsStatus = make(map[string]entities.LightningMonkeyAgentReportStatusItem)
	}
	go a.startStatusTracing()
	if a.statusLock == nil {
		a.statusLock = &sync.RWMutex{}
	}
	if a.recoveryLock == nil {
		a.recoveryLock = &sync.Mutex{}
	}
	if a.handlerFactory == nil {
		a.handlerFactory = &AgentJobHandlerFactory{}
		a.handlerFactory.Initialize(a.c, a)
	}
	if a.workQueue == nil {
		a.workQueue = make(chan *entities.AgentJob, 1)
	}
	c, err := client.NewEnvClient()
	if err != nil {
		logrus.Fatalf("Failed to initialize docker client, error: %s %w", err.Error(), crashError)
		return
	}
	a.dockerClient = c
}

func (a *LightningMonkeyAgent) startStatusTracing() {
	for {
		select {
		case rs, isOpen := <-a.c:
			if !isOpen {
				return
			}
			a.statusLock.Lock()
			a.ItemsStatus[rs.Key] = rs.Item
			a.statusLock.Unlock()
			a.updateRecoveryFile(rs)
		default:
			//NOP
			time.Sleep(time.Second * 3)
		}
		//for debugging only.
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			a.statusLock.RLock()
			for k, v := range a.ItemsStatus {
				logrus.Debugf("Role: %s, Status: %#v", k, v)
			}
			a.statusLock.RUnlock()
		}
	}
}

func (a *LightningMonkeyAgent) updateRecoveryFile(rs LightningMonkeyAgentReportStatus) {
	//not fully initialized, waiting...
	if a.rr == nil {
		return
	}
	var err error
	switch rs.Key {
	case entities.AgentJob_Deploy_ETCD:
		if !a.rr.HasInstalledETCD && rs.Item.HasProvisioned {
			a.rr.HasInstalledETCD = true
			a.rr.ETCDDeploymentType = COMPONENT_DEPLOYMENT_INTEGRATION
			a.rr.InstallETCDTime = rs.Item.LastSeenTime
			logrus.Debugf("Try writing recovery file with ETCD deployment status...")
			if err = a.saveRecoveryFile(); err != nil {
				logrus.Error(err.Error())
			}
		}
	case entities.AgentJob_Deploy_Master:
		if !a.rr.HasInstalledMaster && rs.Item.HasProvisioned {
			a.rr.HasInstalledMaster = true
			a.rr.MasterDeploymentType = COMPONENT_DEPLOYMENT_INTEGRATION
			a.rr.InstallMasterTime = rs.Item.LastSeenTime
			logrus.Debugf("Try writing recovery file with Kubernetes master deployment status...")
			if err = a.saveRecoveryFile(); err != nil {
				logrus.Error(err.Error())
			}
		}
	case entities.AgentJob_Deploy_Minion:
		if !a.rr.HasInstalledMinion && rs.Item.HasProvisioned {
			a.rr.HasInstalledMinion = true
			a.rr.MinionDeploymentType = COMPONENT_DEPLOYMENT_INTEGRATION
			a.rr.InstallMinionTime = rs.Item.LastSeenTime
			logrus.Debugf("Try writing recovery file with Kubernetes minion deployment status...")
			if err = a.saveRecoveryFile(); err != nil {
				logrus.Error(err.Error())
			}
		}
	default:
		//NOP
	}
}

func (a *LightningMonkeyAgent) Start() {
	var err error
	//try to recover system before fully start it up.
	if err = a.recover(); err != nil {
		logrus.Fatalf("Occurred serious problem during recovery procedure, error: %s", err.Error())
		return
	}
	//recovered from file.
	if a.rr != nil {
		a.arg.ClusterId = &a.rr.ClusterID
		a.arg.IsETCDRole = &a.rr.IsETCDRole
		a.arg.IsMasterRole = &a.rr.IsMasterRole
		a.arg.IsMinionRole = &a.rr.IsMinionRole
		a.arg.IsHARole = &a.rr.IsHARole
	} else {
		//create new recovery record when it's the first time to boot up.
		a.rr = &RecoveryRecord{ClusterID: *a.arg.ClusterId}
	}
	//start new go-routine for periodic reporting its status.
	go a.reportStatus()
	//start new go-routine for performing jobs.
	go a.performJob()
	//main loop start here.
	for {
		time.Sleep(time.Second * 5)
		//try to register itself.
		err = a.Register()
		if err != nil {
			if err == errNotInitialized {
				logrus.Infof("Entering resource pool mode, waiting for the remote call...")
				continue
			}
			logrus.Errorf("Failed to register to API server, error: %s", err.Error())
			if xerrors.Is(err, crashError) {
				os.Exit(1)
			}
			continue
		}
		job, err := a.queryJob()
		if err != nil {
			logrus.Errorf("Failed to query job to API server, error: %s", err.Error())
			if xerrors.Is(err, crashError) {
				os.Exit(1)
			}
			continue
		}
		if job == nil {
			continue
		}
		if job.Name == entities.AgentJob_NOP {
			logrus.Info(job.Reason)
			continue
		}
		//do block when it's busy performing previous job.
		a.workQueue <- job
		//start tracing current executing job progressing.
		a.currentJob = job
		for {
			time.Sleep(time.Second)
			if a.currentJob == nil {
				break
			}
			if a.currentJob.HadDone {
				a.currentJob = nil
				break
			}
			if a.currentJob.HealthCheckHandler != nil {
				hc := a.currentJob.HealthCheckHandler.(AgentJobHandler)
				healthy, err := hc(a.currentJob, a)
				if err != nil {
					logrus.Errorf("Waiting, Failed to check job(%s)'s health status, error: %s", a.currentJob.Name, err.Error())
					continue
				}
				if healthy {
					a.currentJob = nil
					break
				}
			}
			logrus.Debugf("Waiting for job execution finish...Job: %s, Args: %#v", a.currentJob.Name, a.currentJob.Arguments)
		}
	}
}

func (a *LightningMonkeyAgent) reportStatus() {
	var err error
	for {
		time.Sleep(time.Second * 3)
		if atomic.LoadInt32(&a.hasRegistered) == 0 {
			continue
		}
		err = a.reportStatusInternal()
		if err != nil {
			logrus.Errorf("Failed to report status to API server, error: %s", err.Error())
			if xerrors.Is(err, crashError) {
				os.Exit(1)
			}
		}
	}
}

func (a *LightningMonkeyAgent) reportStatusInternal() error {
	client := http.Client{
		Timeout:   time.Second * 5,
		Transport: http.DefaultTransport,
	}
	status := entities.LightningMonkeyAgentReportStatus{
		IP:      *a.arg.Address,
		LeaseId: a.arg.LeaseId,
		Items:   a.cloneStatusMap(),
	}
	bodyData, err := json.Marshal(status)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/apis/v1/agent/status?agent-id=%s&cluster-id=%s", *a.arg.Server, a.arg.AgentId, *a.arg.ClusterId), bytes.NewBuffer(bodyData))
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	rsp, err := client.Do(req)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	defer rsp.Body.Close()
	obj := entities.AgentReportStatusResponse{}
	rspData, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	err = json.Unmarshal(rspData, &obj)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	if obj.ErrorId != entities.Succeed {
		internalErr := fmt.Errorf("Failed to report status remote API server, biz code: %d, error: %s", obj.ErrorId, obj.Reason)
		if !obj.NeedCrash {
			return internalErr
		}
		return xerrors.Errorf("%s %w", internalErr.Error(), crashError)
	}
	//reset lease-id for avoiding lease not working problems. (re-connecting after over allowed maximum heart-beat interval)
	a.arg.LeaseId = obj.LeaseId
	return nil
}

func (a *LightningMonkeyAgent) cloneStatusMap() map[string]entities.LightningMonkeyAgentReportStatusItem {
	a.statusLock.RLock()
	defer a.statusLock.RUnlock()
	sm := make(map[string]entities.LightningMonkeyAgentReportStatusItem)
	for k, v := range a.ItemsStatus {
		sm[k] = v
	}
	return sm
}

func (a *LightningMonkeyAgent) queryJob() (*entities.AgentJob, error) {
	client := http.Client{
		Timeout:   time.Second * 5,
		Transport: http.DefaultTransport,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apis/v1/agent/query?agent-id=%s&cluster-id=%s", *a.arg.Server, a.arg.AgentId, *a.arg.ClusterId), nil)
	if err != nil {
		return nil, err
	}
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	obj := entities.GetNextAgentJobResponse{}
	rspData, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(rspData, &obj)
	if err != nil {
		return nil, err
	}
	if obj.ErrorId != entities.Succeed {
		internalErr := fmt.Errorf("Failed to query job from remote API server, biz code: %d, error: %s", obj.ErrorId, obj.Reason)
		if !obj.NeedCrash {
			return nil, internalErr
		}
		return nil, xerrors.Errorf("%s %w", internalErr.Error(), crashError)
	}
	return obj.Job, nil
}

func (a *LightningMonkeyAgent) performJob() {
	var err error
	var job *entities.AgentJob
	var isOpen bool
	for {
		job, isOpen = <-a.workQueue
		if !isOpen {
			job.HadDone = true
			return
		}
		err = a.handleJob(job)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func (a *LightningMonkeyAgent) handleJob(job *entities.AgentJob) error {
	var err error
	var succeed bool
	var handlers []AgentJobHandler
	handlers = a.handlerFactory.GetHandler(job.Name)
	if handlers == nil {
		job.HadDone = true
		logrus.Fatalf("No any handler could process this job: %s", job.Name)
		return nil
	}
	job.HealthCheckHandler = handlers[1]
	succeed, err = handlers[1](job, a)
	if err != nil {
		job.HadDone = true
		return fmt.Errorf("Failed to process job(Phase -> Health Check): %#v, error: %s", job, err.Error())
	}
	if succeed {
		job.HadDone = true
		logrus.Debugf("Skipped job: %s, It's already running...", job.Name)
		return nil
	}
	//do provision.
	succeed, err = handlers[0](job, a)
	if err != nil {
		job.HadDone = true
		if xerrors.Is(err, crashError) {
			os.Exit(1)
		}
		return fmt.Errorf("Failed to process job: %#v, error: %s", job, err.Error())
	}
	if !succeed {
		job.HadDone = true
		return fmt.Errorf("Failed to process job: %#v, which returned an un-successful status!", job)
	}
	return nil
}

func (a *LightningMonkeyAgent) runKubeletContainer(masterIP string) error {
	var err error
	var cs []types.Container
	cs, err = a.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return xerrors.Errorf("Failed to get running container list, error: %s %w", err.Error(), crashError)
	}
	//check whether a container named "kubelet" has been started.
	if cs != nil && len(cs) > 0 {
		for i := 0; i < len(cs); i++ {
			if cs[i].Names[0] == "/kubelet" {
				if strings.Contains(cs[i].Status, "Up") {
					//kubelet has been started successfully, skip other actions.
					return nil
				} else {
					return xerrors.Errorf("\"kubelet\" has been started but with unhealthy container status %w", crashError)
				}
			}
		}
	}
	if masterIP == "" {
		err = k8s.GenerateKubeletConfig(CERTIFICATE_STORAGE_PATH, *a.arg.Address, a.masterSettings)
	} else {
		err = k8s.GenerateKubeletConfig(CERTIFICATE_STORAGE_PATH, masterIP, a.masterSettings)
	}
	if err != nil {
		return xerrors.Errorf("Failed to generate kube-config, master-ip: %s, error: %s %w", masterIP, err.Error(), crashError)
	}
	img := a.basicImages.Images["k8s"].ImageName
	infraContainer := a.basicImages.Images["infra"].ImageName

	//--volume=/:/rootfs:ro
	//--volume=/sys:/sys:ro
	//--volume=/dev:/dev
	//--volume=/var/lib/docker/:/var/lib/docker:rw
	//--volume=/var/lib/kubelet/:/var/lib/kubelet:shared
	//--volume=/var/run:/var/run:rw
	cmd := []string{
		"kubelet",
		fmt.Sprintf("--config=%s", filepath.Join(CERTIFICATE_STORAGE_PATH, "kubelet_settings.yml")),
		//fmt.Sprintf("--bootstrap-kubeconfig=%s", filepath.Join(CERTIFICATE_STORAGE_PATH, "bootstrap-kubelet.conf")),
		fmt.Sprintf("--kubeconfig=%s", filepath.Join(CERTIFICATE_STORAGE_PATH, "kubelet.conf")),
		fmt.Sprintf("--pod-infra-container-image=%s", infraContainer),
		fmt.Sprintf("--register-node=%t", *a.arg.IsMinionRole),
		fmt.Sprintf("--hostname-override=%s", *a.arg.Address),
		"--cgroup-driver=cgroupfs",
		"--cgroups-per-qos=false",
		"--enforce-node-allocatable=",
		"--allow-privileged=true",
		"--network-plugin=cni",
		"--serialize-image-pulls=false",
		//"--address=0.0.0.0",
	}
	if a.arg.NodeLabels != nil && *a.arg.NodeLabels != "" {
		cmd = append(cmd, fmt.Sprintf("--node-labels=%s", *a.arg.NodeLabels))
	}
	binds := []string{
		//"/:/rootfs:ro",
		"/sys:/sys:ro",
		//"/dev:/dev",
		"/etc:/etc",
		"/var/run:/var/run:rw",
		"/var/lib/docker:/var/lib/docker:rw",
		"/var/lib/kubelet:/var/lib/kubelet:rshared",
		"/opt/cni/bin:/opt/cni/bin",
	}
	if v, isOK := a.masterSettings[entities.MasterSettings_DockerExtraGraphPath]; isOK && v != "" {
		binds = append(binds, fmt.Sprintf("%s:%s:rw", v, v))
	}
	resp, err := a.dockerClient.ContainerCreate(context.Background(), &container.Config{
		Hostname: *a.arg.Address,
		Image:    img,
		Tty:      false,
		Cmd:      cmd,
		Volumes:  map[string]struct{}{},
	}, &container.HostConfig{
		Binds:         binds,
		Privileged:    true,
		NetworkMode:   "host",
		PidMode:       "host",
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}, &network.NetworkingConfig{}, "kubelet")
	if err != nil {
		return xerrors.Errorf("Failed to create container, error: %s %w", err.Error(), crashError)
	}
	if err = a.dockerClient.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return xerrors.Errorf("Failed to start container, error: %s %w", err.Error(), crashError)
	}
	out, err := a.dockerClient.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return xerrors.Errorf("Failed to retrieve container logs, error: %s %w", err.Error(), crashError)
	}
	_, _ = io.Copy(os.Stdout, out)
	return nil
}

//hasInitializedRoles returns false when it is waiting for the remote call.
func (a *LightningMonkeyAgent) hasInitializedRoles() bool {
	return *a.arg.IsETCDRole || *a.arg.IsMasterRole || *a.arg.IsMinionRole || *a.arg.IsHARole
}
