package main

import (
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"time"
)

func (a *LightningMonkeyAgent) recover() error {
	var err error
	if _, err = os.Stat(RECOVERY_FILE_PATH); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("Failed to scan recovery file, error: %s", err.Error())
		}
		return nil
	}
	data, err := ioutil.ReadFile(RECOVERY_FILE_PATH)
	if err != nil {
		return err
	}
	var rr RecoveryRecord
	err = json.Unmarshal(data, &rr)
	if err != nil {
		return err
	}
	a.rr = &rr
	logrus.Warn("Entering recovery mode...")
	//wait until all of installed components becomes healthy.
	for {
		if a.checkHealthy() {
			break
		}
		time.Sleep(time.Second * 3)
	}
	logrus.Warn("Exited recovery mode!")
	return nil
}

func (a *LightningMonkeyAgent) checkHealthy() bool {
	a.statusLock.RLock()
	defer a.statusLock.RUnlock()
	var isOK bool
	var rs entities.LightningMonkeyAgentReportStatusItem
	if a.rr.HasInstalledETCD {
		if rs, isOK = a.ItemsStatus[entities.AgentJob_Deploy_ETCD]; !isOK || !rs.HasProvisioned {
			logrus.Debugf("[RECOVERY MODE] Waiting...ETCD still not healthy!")
			return false
		}
	}
	if a.rr.HasInstalledMaster {
		if rs, isOK = a.ItemsStatus[entities.AgentJob_Deploy_Master]; !isOK || !rs.HasProvisioned {
			logrus.Debugf("[RECOVERY MODE] Waiting...Kubernetes master components still not healthy!")
			return false
		}
	}
	if a.rr.HasInstalledMinion {
		if rs, isOK = a.ItemsStatus[entities.AgentJob_Deploy_Minion]; !isOK || !rs.HasProvisioned {
			logrus.Debugf("[RECOVERY MODE] Waiting...Kubernetes minion components still not healthy!")
			return false
		}
	}
	return true
}

func (a *LightningMonkeyAgent) saveRecoveryFile() error {
	return nil
}
