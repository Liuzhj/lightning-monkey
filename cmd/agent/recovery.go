package main

import (
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
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
	a.recoveryLock.Lock()
	defer a.recoveryLock.Unlock()
	_ = os.Remove(RECOVERY_FILE_PATH)
	//create path.
	err := os.MkdirAll(path.Dir(RECOVERY_FILE_PATH), 0644) //rw-r--r--
	if err != nil {
		return fmt.Errorf("Failed to create path to save recovery file, error: %s", err.Error())
	}
	data, err := json.Marshal(a.rr)
	if err != nil {
		return fmt.Errorf("Failed to marshal recovery object structure to JSON data, error: %s", err.Error())
	}
	err = ioutil.WriteFile(RECOVERY_FILE_PATH, data, 0644) //rw-r--r--
	if err != nil {
		return fmt.Errorf("Failed to save recovery file, error: %s", err.Error())
	}
	return nil
}
