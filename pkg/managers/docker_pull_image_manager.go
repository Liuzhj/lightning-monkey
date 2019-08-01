package managers

import (
	"context"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"io"
	"os"
)

type RemoteRegistryDockerImageManager struct {
	dockerClient            *client.Client
	imageCollectionSettings *entities.DockerImageCollection
}

func (im *RemoteRegistryDockerImageManager) Ready() error {
	if im.imageCollectionSettings.Images == nil || len(im.imageCollectionSettings.Images) == 0 {
		return nil
	}
	for _, v := range im.imageCollectionSettings.Images {
		logrus.Infof("Pulling docker image: %s", v.ImageName)
		reader, err := im.dockerClient.ImagePull(context.Background(), v.ImageName, types.ImagePullOptions{})
		if err != nil {
			return xerrors.Errorf("Failed to pull docker image, error: %s %w", err.Error(), crashError)
		}
		_, _ = io.Copy(os.Stdout, reader)
	}
	return nil
}
