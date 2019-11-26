package built_in

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/g0194776/lightningmonkey/pkg/k8s"
	"github.com/g0194776/lightningmonkey/pkg/utils"
	"github.com/sirupsen/logrus"
	ko "k8s.io/api/core/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sync/atomic"
)

type DefaultImagePullingSecretsDeploymentController struct {
	client        *k8s.KubernetesClientSet
	settings      entities.LightningMonkeyClusterSettings
	parsedObjects []runtime.Object
	hasInstalled  int32
}

func (dc *DefaultImagePullingSecretsDeploymentController) Initialize(client *k8s.KubernetesClientSet, clientIp string, settings entities.LightningMonkeyClusterSettings) error {
	dc.client = client
	dc.settings = settings
	if dc.settings.ImagePullSecrets != nil && len(dc.settings.ImagePullSecrets) > 0 {
		for i := 0; i < len(dc.settings.ImagePullSecrets); i++ {
			s := dc.settings.ImagePullSecrets[i]
			secret := newSecret(s)
			dc.parsedObjects = append(dc.parsedObjects, &secret)
		}
	}
	return nil
}

func newSecret(s entities.ImagePullSecret) ko.Secret {
	var user, pass, email string
	user = s.Username
	pass = s.Password
	email = s.Email
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, pass)))
	child2Map := map[string]interface{}{
		"username": user,
		"password": pass,
		"email":    email,
		"auth":     auth,
	}
	childMap := map[string]interface{}{
		s.Registry: child2Map,
	}
	rootMap := map[string]interface{}{}
	rootMap["auths"] = childMap
	data, err := json.Marshal(rootMap)
	if err != nil {
		panic(err)
	}
	return ko.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Type: ko.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(base64.StdEncoding.EncodeToString(data)),
		},
	}
}

func (dc *DefaultImagePullingSecretsDeploymentController) Install() error {
	if dc.parsedObjects == nil || len(dc.parsedObjects) == 0 {
		return nil
	}
	var err error
	var existed bool
	var hasInstalled bool
	hasInstalled, err = dc.HasInstalled()
	if err != nil {
		return fmt.Errorf("Failed to check installation status in the %s deployment controller, error: %s", dc.GetName(), err.Error())
	}
	//duplicated installation action, ignore.
	if hasInstalled {
		return nil
	}
	logrus.Infof("Start provisioning %s for cluster: %s", dc.GetName(), dc.settings.Id)
	for i := 0; i < len(dc.parsedObjects); i++ {
		metadata, err := utils.ObjectMetaFor(dc.parsedObjects[i])
		if err != nil {
			return fmt.Errorf("Failed to get Kubernetes resource, error: %s", err.Error())
		}
		if existed, err = k8s.IsKubernetesResourceExists(dc.client, dc.parsedObjects[i]); err != nil && !k8sErr.IsNotFound(err) {
			return fmt.Errorf("Failed to check Kubernetes resource existence, error: %s", err.Error())
		} else if !existed {
			_, err = k8s.CreateK8SResource(dc.client, dc.parsedObjects[i])
			if err != nil {
				return fmt.Errorf("Failed to create Kubernetes resource: %s, error: %s", metadata.Name, err.Error())
			}
		}
		logrus.Infof("Kubernetes resource %s(%s) has been created successfully!", metadata.Name, dc.parsedObjects[i].GetObjectKind().GroupVersionKind().Kind)
	}
	return nil
}

func (dc *DefaultImagePullingSecretsDeploymentController) UnInstall() error {
	panic("implement me")
}

func (dc *DefaultImagePullingSecretsDeploymentController) GetName() string {
	return "Image Pulling Secrets"
}

func (dc *DefaultImagePullingSecretsDeploymentController) HasInstalled() (installed bool, err error) {
	if atomic.LoadInt32(&dc.hasInstalled) == 1 {
		return true, nil
	}
	defer func() {
		if installed {
			atomic.StoreInt32(&dc.hasInstalled, 1)
		}
	}()
	if dc.settings.ExtensionalDeployments == nil || len(dc.settings.ExtensionalDeployments) == 0 {
		//skipping installation procedure.
		return true, nil
	}
	if dc.settings.ImagePullSecrets == nil || len(dc.settings.ImagePullSecrets) == 0 {
		return true, nil
	}
	//only validate top one of given secret collection.
	s, err := dc.client.CoreClient.CoreV1().Secrets(dc.settings.ImagePullSecrets[0].Namespace).Get(dc.settings.ImagePullSecrets[0].Name, v1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed to retrieve Secret(%s/%s) object from given Kubernetes cluster, error: %s", dc.settings.ImagePullSecrets[0].Namespace, dc.settings.ImagePullSecrets[0].Name, err.Error())
	}
	return s != nil, nil
}
