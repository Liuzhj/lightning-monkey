package main

import (
	"bytes"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type LightningMonkeyAgent struct {
	arg                *AgentArgs
	lastRegisteredTime time.Time
	lastReportTime     time.Time
	hasRegistered      int32
	workQueue          chan *entities.AgentJob
	handlerFactory     *AgentJobHandlerFactory
}

var (
	crashError = errors.New("CRASH ERROR")
)

func (a *LightningMonkeyAgent) Register() (err error) {
	if atomic.LoadInt32(&a.hasRegistered) == 1 {
		return nil
	}
	defer func() {
		if err == nil {
			if atomic.CompareAndSwapInt32(&a.hasRegistered, 0, 1) {
				a.lastRegisteredTime = time.Now()
			}
		} else {
			atomic.SwapInt32(&a.hasRegistered, 0)
		}
	}()
	clusterId := bson.ObjectIdHex(*a.arg.ClusterId)
	hostname, _ := os.Hostname()
	agentObj := entities.Agent{
		HasETCDRole:   *a.arg.IsETCDRole,
		HasMasterRole: *a.arg.IsMasterRole,
		HasMinionRole: *a.arg.IsMinionRole,
		MetadataId:    *a.arg.MetadataId,
		ClusterId:     &clusterId,
		Hostname:      hostname,
	}
	bodyData, err := json.Marshal(agentObj)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	client := http.Client{
		Timeout:   time.Second * 5,
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
	rspObj := entities.Response{}
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
	return a.downloadCertificates()
}

func (a *LightningMonkeyAgent) downloadCertificates() error {
	err := os.MkdirAll(CERTIFICATE_STORAGE_PATH, 0664) //"rw-rw-r--"
	if err != nil {
		return xerrors.Errorf("Failed to create certificate storage path: %s %w", err.Error(), crashError)
	}
	neededCerts := []string{"ca.crt", "ca.key", "etcd/ca.crt", "etcd/ca.key"}
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
	f, err := os.Create(filePath)
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
	if a.handlerFactory == nil {
		a.handlerFactory = &AgentJobHandlerFactory{}
		a.handlerFactory.Initialize()
	}
	if a.workQueue == nil {
		a.workQueue = make(chan *entities.AgentJob, 1)
	}
}

func (a *LightningMonkeyAgent) Start() {
	var err error
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
	status := entities.AgentStatus{Status: entities.AgentStatus_Running}
	bodyData, err := json.Marshal(status)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/apis/v1/agent/status?metadata-id=%s", *a.arg.Server, *a.arg.MetadataId), bytes.NewBuffer(bodyData))
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	rsp, err := client.Do(req)
	if err != nil {
		return xerrors.Errorf("%s %w", err.Error(), crashError)
	}
	defer rsp.Body.Close()
	obj := entities.Response{}
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
	return nil
}

func (a *LightningMonkeyAgent) queryJob() (*entities.AgentJob, error) {
	client := http.Client{
		Timeout:   time.Second * 5,
		Transport: http.DefaultTransport,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apis/v1/agent/query?metadata-id=%s", *a.arg.Server, *a.arg.MetadataId), nil)
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
	var job *entities.AgentJob
	var handler AgentJobHandler
	var err error
	var isOpen bool
	for {
		job, isOpen = <-a.workQueue
		if !isOpen {
			return
		}
		handler = a.handlerFactory.GetHandler(job.Name)
		if handler == nil {
			logrus.Warnf("No any handler could process this job: %s", job.Name)
			continue
		}
		err = handler(job)
		if err != nil {
			logrus.Errorf("Failed to process job: %#v, error: %s", job, err.Error())
			if xerrors.Is(err, crashError) {
				os.Exit(1)
			}
		}
	}
}
