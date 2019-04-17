package certs

import (
	"bufio"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

type GeneratedCertsMap struct {
	res  map[string]string
	path string
}

func (gc *GeneratedCertsMap) InitializeData(data map[string]string) *GeneratedCertsMap {
	gc.res = data
	return gc
}

func (gm *GeneratedCertsMap) CleanResource() {
	if _, err := os.Stat(gm.path); err != nil && os.IsExist(err) {
		_ = os.RemoveAll(gm.path)
	}
}

func (gm *GeneratedCertsMap) GetResources() map[string]string {
	return gm.res
}

func GenerateAdminKubeConfig(advertiseAddr string, basicCertMap entities.CertificateCollection) (*GeneratedCertsMap, error) {
	path := fmt.Sprintf("/tmp/kubernetes-certs/%s", uuid.NewV4().String())
	err := os.MkdirAll(path, 0644)
	if err != nil {
		return nil, fmt.Errorf("Failed to create temporary path: %s, error: %s", path, err.Error())
	}
	logrus.Infof("Kube admin configuration file temporary storage path: %s", path)
	defer func() {
		//remove entire path.
		_ = os.RemoveAll(path)
	}()
	if basicCertMap != nil {
		for i := 0; i < len(basicCertMap); i++ {
			_ = os.MkdirAll(filepath.Dir(filepath.Join(path, basicCertMap[i].Name)), 0644)
			ioErr := ioutil.WriteFile(filepath.Join(path, basicCertMap[i].Name), []byte(basicCertMap[i].Content), 0644)
			if ioErr != nil {
				return nil, fmt.Errorf("Failed to save basic certificate(%s) to path: %s, error: %s", basicCertMap[i].Name, path, ioErr.Error())
			}
		}
	}
	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("kubeadm init phase kubeconfig admin --cert-dir=%s --kubeconfig-dir=%s --apiserver-advertise-address=%s", path, path, advertiseAddr))
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
	return getCertificatesContent(path, "", nil)
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
	return getCertificatesContent(path, "", nil)
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
	certMap, err := getCertificatesContent(path, "", nil)
	if err != nil {
		return nil, err
	}
	//ETCD ca.
	cmd = exec.Command("/bin/bash", "-c", fmt.Sprintf("kubeadm init phase certs etcd-ca --cert-dir=%s", path))
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	reader = bufio.NewReader(stdout)
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
	return getCertificatesContent(path, "etcd", certMap)
}

func GenerateETCDClientCertificatesAndManifest(certPath, etcdConfigContent string) error {
	configFilePath := filepath.Join(certPath, "etcd_config.yml")
	_ = os.RemoveAll(configFilePath)
	f, err := os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(etcdConfigContent)
	if err != nil {
		return err
	}
	subCommands := []string{
		"kubeadm init phase certs etcd-server",
		"kubeadm init phase certs etcd-peer",
		"kubeadm init phase certs etcd-healthcheck-client",
		"kubeadm init phase certs apiserver-etcd-client",
		"kubeadm init phase etcd local",
	}
	var cmd *exec.Cmd
	for i := 0; i < len(subCommands); i++ {
		cmd = exec.Command("/bin/bash", "-c", fmt.Sprintf("%s --config=%s", subCommands[i], configFilePath))
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err = cmd.Start(); err != nil {
			return err
		}
		reader := bufio.NewReader(stdout)
		for {
			traceData, _, err := reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					return err
				}
				break
			}
			logrus.Infof(string(traceData))
		}
		if err = cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func GenerateMasterCertificatesAndManifest(certPath, address string, settings map[string]string) error {
	subCommands := []string{
		fmt.Sprintf("kubeadm init phase certs apiserver --apiserver-advertise-address=%s --service-dns-domain=%s --service-cidr=%s --cert-dir=%s",
			address,
			settings[entities.MasterSettings_ServiceDNSDomain],
			settings[entities.MasterSettings_ServiceCIDR],
			certPath,
		),
		fmt.Sprintf("kubeadm init phase certs apiserver-etcd-client --cert-dir=%s", certPath),
		fmt.Sprintf("kubeadm init phase certs apiserver-kubelet-client --cert-dir=%s", certPath),
		fmt.Sprintf("kubeadm init phase certs sa --cert-dir=%s", certPath),
		fmt.Sprintf("kubeadm init phase certs front-proxy-ca --cert-dir=%s", certPath),
		fmt.Sprintf("kubeadm init phase certs front-proxy-client --cert-dir=%s", certPath),
		fmt.Sprintf("kubeadm init phase kubeconfig controller-manager --apiserver-advertise-address=%s --cert-dir=%s", address, certPath),
		fmt.Sprintf("kubeadm init phase kubeconfig scheduler --apiserver-advertise-address=%s --cert-dir=%s", address, certPath),
		fmt.Sprintf("kubeadm init phase control-plane all --apiserver-advertise-address=%s --kubernetes-version=%s --pod-network-cidr=%s --service-cidr=%s --cert-dir=%s",
			address, //address,
			settings[entities.MasterSettings_KubernetesVersion],
			settings[entities.MasterSettings_PodCIDR],
			settings[entities.MasterSettings_ServiceCIDR],
			certPath,
		),
	}
	var cmd *exec.Cmd
	for i := 0; i < len(subCommands); i++ {
		cmd = exec.Command("/bin/bash", "-c", subCommands[i])
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err = cmd.Start(); err != nil {
			return err
		}
		reader := bufio.NewReader(stdout)
		for {
			traceData, _, err := reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					return err
				}
				break
			}
			logrus.Infof(string(traceData))
		}
		if err = cmd.Wait(); err != nil {
			return err
		}
	}
	//replace used docker registry.
	manifestPath := filepath.Join(certPath, "../", "manifests")
	logrus.Infof("Calculated manifest file path: %s", manifestPath)
	err := filepath.Walk(manifestPath, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		f, e := os.OpenFile(p, os.O_RDWR, 0600)
		if e != nil {
			return e
		}
		defer f.Close()
		fileData, e := ioutil.ReadAll(f)
		if e != nil {
			return e
		}
		e = f.Truncate(0)
		if e != nil {
			return e
		}
		_, e = f.Seek(0, 0)
		if e != nil {
			return e
		}
		//replace docker registry.
		re := regexp.MustCompile(`k8s.gcr.io/kube-apiserver|k8s.gcr.io/kube-scheduler|k8s.gcr.io/kube-controller-manager`)
		newContent := re.ReplaceAllString(string(fileData), settings[entities.MasterSettings_DockerRegistry])
		//replace kube-controller-manager & scheduler settings.
		re = regexp.MustCompile(`(--address=)(.*)`)
		newContent = re.ReplaceAllString(newContent, "${1}0.0.0.0")
		//replace ETCD settings.
		re = regexp.MustCompile(`(etcd-servers=)(.*)`)
		newContent = re.ReplaceAllString(newContent, fmt.Sprintf("${1}https://%s:2379", address))
		re = regexp.MustCompile(`(advertise-address=)(.*)`)
		newContent = re.ReplaceAllString(newContent, fmt.Sprintf("${1}%s", address))
		re = regexp.MustCompile(`(host:)(.*)`)
		newContent = re.ReplaceAllString(newContent, fmt.Sprintf("${1} %s", address))
		_, e = f.Write([]byte(newContent))
		if e != nil {
			return e
		}
		return f.Sync()
	})
	return err
}

func getCertificatesContent(path, title string, m *GeneratedCertsMap) (*GeneratedCertsMap, error) {
	if m == nil {
		m = &GeneratedCertsMap{res: make(map[string]string), path: path}
	}
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
		if title != "" {
			m.res[filepath.Join(title, info.Name())] = string(fileData)
		} else {
			m.res[info.Name()] = string(fileData)
		}
		return nil
	})
	return m, err
}
