package certs

import (
	"bufio"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type GeneratedCertsMap struct {
	res  map[string]string
	path string
}

func (gm *GeneratedCertsMap) CleanResource() {
	if _, err := os.Stat(gm.path); err != nil && os.IsExist(err) {
		_ = os.RemoveAll(gm.path)
	}
}

func (gm *GeneratedCertsMap) GetResources() map[string]string {
	return gm.res
}

func GenerateMasterCertificates(advertiseAddr, serviceCIDR string) (*GeneratedCertsMap, error) {
	path := fmt.Sprintf("/tmp/kubernetes-certs/%s", uuid.NewV4().String())
	logrus.Infof("Certificates temporary storage path: %s", path)
	defer func() {
		//remove certs path.
		_ = os.RemoveAll(path)
	}()
	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("kubeadm init phase certs all --cert-dir=%s --apiserver-advertise-address=%s --service-cidr=%s", path, advertiseAddr, serviceCIDR))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(stdout)
	for {
		traceData, _, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		logrus.Infof(string(traceData))
	}
	if err = cmd.Wait(); err != nil {
		return nil, err
	}
	return getCertificatesContent(path)
}

func GenerateMainCACertificates() (*GeneratedCertsMap, error) {
	path := fmt.Sprintf("/tmp/kubernetes-certs/%s", uuid.NewV4().String())
	logrus.Infof("Certificates temporary storage path: %s", path)
	defer func() {
		//remove certs path.
		_ = os.RemoveAll(path)
	}()
	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("kubeadm init phase certs ca --cert-dir=%s", path))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(stdout)
	for {
		traceData, _, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		logrus.Infof(string(traceData))
	}
	if err = cmd.Wait(); err != nil {
		return nil, err
	}
	return getCertificatesContent(path)
}

func getCertificatesContent(path string) (*GeneratedCertsMap, error) {
	m := GeneratedCertsMap{res: make(map[string]string), path: path}
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		f, err := os.OpenFile(p, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		fileData, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		m.res[info.Name()] = string(fileData)
		return nil
	})
	return &m, err
}
