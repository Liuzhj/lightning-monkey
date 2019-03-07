package main

import (
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"os"
	"time"
)

type LightningMonkeyAgent struct {
	arg                *AgentArgs
	lastRegisteredTime time.Time
	lastReportTime     time.Time
	hasRegistered      bool
	workQueue          chan *entities.AgentJob
	handlerFactory     *AgentJobHandlerFactory
}

var (
	crashError = errors.New("CRASH ERROR")
)

func (a *LightningMonkeyAgent) Register() error {
	if a.hasRegistered {
		return nil
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
		//do block when it's busy performing previous job.
		a.workQueue <- job
	}
}

func (a *LightningMonkeyAgent) reportStatus() {
	for {
		time.Sleep(time.Second * 3)
		if !a.hasRegistered {
			continue
		}
	}
}

func (a *LightningMonkeyAgent) queryJob() (*entities.AgentJob, error) {
	client := http.Client{
		Timeout:   time.Second * 5,
		Transport: http.DefaultTransport,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/apis/v1/agent/query?metadata-id=%s", a.arg.Server, a.arg.MetadataId), nil)
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
	for {
		job = <-a.workQueue
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
