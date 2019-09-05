package certs

import (
	"bufio"
	"errors"
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
	"strings"
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

//go:generate mockgen -package=mock_lm -destination=../../mocks/mock_cert_manager.go -source=k8s_certs_generator.go CertificateManager
type CertificateManager interface {
	GenerateAdminKubeConfig(advertiseAddr string, basicCertMap entities.LightningMonkeyCertificateCollection) (*GeneratedCertsMap, error)
	GenerateMasterCertificates(advertiseAddr, serviceCIDR string) (*GeneratedCertsMap, error)
	GenerateMainCACertificates() (*GeneratedCertsMap, error)
	GenerateETCDClientCertificatesAndManifest(certPath, etcdConfigContent string) error
	GenerateMasterCertificatesAndManifest(certPath, address string, settings map[string]string, imageCollection *entities.DockerImageCollection) error
}

type CertificateManagerImple struct {
}

func (cm *CertificateManagerImple) GenerateAdminKubeConfig(advertiseAddr string, basicCertMap entities.LightningMonkeyCertificateCollection) (*GeneratedCertsMap, error) {
	if basicCertMap == nil || len(basicCertMap) == 0 {
		return nil, errors.New("Failed to generate kube-admin config without any basic certificates!")
	}
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
			logrus.Infof("Pareparing write cert: %s", basicCertMap[i].Name)
			_ = os.MkdirAll(filepath.Dir(filepath.Join(path, basicCertMap[i].Name)), 0644)
			ioErr := ioutil.WriteFile(filepath.Join(path, basicCertMap[i].Name), []byte(basicCertMap[i].Value), 0644)
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

func (cm *CertificateManagerImple) GenerateMasterCertificates(advertiseAddr, serviceCIDR string) (*GeneratedCertsMap, error) {
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

func (cm *CertificateManagerImple) GenerateMainCACertificates() (*GeneratedCertsMap, error) {
	path := fmt.Sprintf("/tmp/kubernetes-certs/%s", uuid.NewV4().String())
	logrus.Infof("Certificates temporary storage path: %s", path)
	defer func() {
		//remove certs path.
		_ = os.RemoveAll(path)
	}()
	subCommands := []string{
		fmt.Sprintf("kubeadm init phase certs ca --cert-dir=%s", path),
		fmt.Sprintf("kubeadm init phase certs etcd-ca --cert-dir=%s", path),
		fmt.Sprintf("kubeadm init phase certs front-proxy-ca --cert-dir=%s", path),
		fmt.Sprintf("kubeadm init phase certs sa --cert-dir=%s", path),
	}
	var err error
	certMap := &GeneratedCertsMap{res: make(map[string]string), path: path}
	for i := 0; i < len(subCommands); i++ {
		err = executeCommand(subCommands[i], "")
		if err != nil {
			return nil, err
		}
		if strings.Contains(subCommands[i], "etcd") {
			certMap, err = getCertificatesContent(path, "etcd", certMap)
		} else {
			certMap, err = getCertificatesContent(path, "", certMap)
		}
		if err != nil {
			return nil, err
		}
	}
	return certMap, nil
}

func (cm *CertificateManagerImple) GenerateETCDClientCertificatesAndManifest(certPath, etcdConfigContent string) error {
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
	for i := 0; i < len(subCommands); i++ {
		err = executeCommand(subCommands[i], configFilePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cm *CertificateManagerImple) GenerateMasterCertificatesAndManifest(certPath, address string, settings map[string]string, imageCollection *entities.DockerImageCollection) error {
	subCommands := []string{
		fmt.Sprintf("kubeadm init phase certs apiserver --apiserver-advertise-address=%s --service-dns-domain=%s --service-cidr=%s --cert-dir=%s --apiserver-cert-extra-sans=%s",
			address,
			settings[entities.MasterSettings_ServiceDNSDomain],
			settings[entities.MasterSettings_ServiceCIDR],
			certPath,
			//Troy:only test vip ipsan
			"192.168.33.100",
		),
		fmt.Sprintf("kubeadm init phase certs apiserver-etcd-client --cert-dir=%s", certPath),
		fmt.Sprintf("kubeadm init phase certs apiserver-kubelet-client --cert-dir=%s", certPath),
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
	var err error
	for i := 0; i < len(subCommands); i++ {
		err = executeCommand(subCommands[i], "")
		if err != nil {
			return err
		}
	}
	//replace used docker registry.
	manifestPath := filepath.Join(certPath, "../", "manifests")
	logrus.Infof("Calculated manifest file path: %s", manifestPath)
	err = filepath.Walk(manifestPath, func(p string, info os.FileInfo, err error) error {
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
		var re *regexp.Regexp
		var newContent string
		logrus.Debugf("Being replace file: %s", p)
		//replace docker registry.
		if strings.Contains(p, "etcd") {
			logrus.Infof("Ignored replacing docker image for ETCD manifest file: %s", p)
			newContent = string(fileData)
		} else {
			re = regexp.MustCompile(`(image: )(.*)`)
			newContent = re.ReplaceAllString(string(fileData), fmt.Sprintf("${1}%s", imageCollection.Images["k8s"].ImageName))
		}
		//replace kube-apiserver & kube-controller-manager & scheduler settings.
		re = regexp.MustCompile(`(--address=)(.*)`)
		newContent = re.ReplaceAllString(newContent, "${1}0.0.0.0")
		re = regexp.MustCompile(`(--enable-admission-plugins=)(.*)`)
		newContent = re.ReplaceAllString(newContent, "${1}NamespaceLifecycle,NamespaceExists,LimitRanger,ResourceQuota,ServiceAccount")
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
		var key string
		if title != "" {
			key = filepath.Join(title, info.Name())
		} else {
			key = info.Name()
		}
		if _, isOK := m.res[key]; isOK {
			return nil
		}
		f, err := os.OpenFile(p, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}
		//fixed FD leak.
		defer f.Close()
		fileData, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		//append file content only if it not exists.
		m.res[key] = string(fileData)
		return nil
	})
	return m, err
}

func executeCommand(command string, configFilePath string) error {
	var cmd *exec.Cmd
	if configFilePath != "" {
		cmd = exec.Command("/bin/bash", "-c", fmt.Sprintf("%s --config=%s", command, configFilePath))
	} else {
		cmd = exec.Command("/bin/bash", "-c", command)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	//fixed FD leak.
	defer stdout.Close()
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
	return nil
}
