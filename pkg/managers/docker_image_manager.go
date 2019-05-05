package managers

import (
	"errors"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/g0194776/lightningmonkey/pkg/entities"
)

type DockerImageManager interface {
	Ready() error
}

var (
	crashError = errors.New("CRASH ERROR")
)

func NewDockerImageManager(serverAddr string, dockerClient *client.Client, imageCollectionSettings *entities.DockerImageCollection) (DockerImageManager, error) {
	if imageCollectionSettings.DownloadType == entities.DockerImageDownloadType_Registry {
		return &RemoteRegistryDockerImageManager{dockerClient: dockerClient, imageCollectionSettings: imageCollectionSettings}, nil
	}
	if imageCollectionSettings.DownloadType == entities.DockerImageDownloadType_HTTP {
		return &HTTPDockerImageManager{dockerClient: dockerClient, imageCollectionSettings: imageCollectionSettings, serverAddr: serverAddr}, nil
	}
	return nil, fmt.Errorf("Unsupported download type of remote Docker image: %s", imageCollectionSettings.DownloadType)
}
