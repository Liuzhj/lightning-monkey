package managers

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type HTTPDockerImageManager struct {
	serverAddr              string
	dockerClient            *client.Client
	imageCollectionSettings *entities.DockerImageCollection
}

func (im *HTTPDockerImageManager) Ready() error {
	if im.imageCollectionSettings.Images == nil || len(im.imageCollectionSettings.Images) == 0 {
		return nil
	}
	var err error
	var closer io.ReadCloser
	//var rsp types.ImageLoadResponse
	for name, v := range im.imageCollectionSettings.Images {
		downloadUrl := fmt.Sprintf(v.DownloadAddr, im.serverAddr, im.imageCollectionSettings.HTTPDownloadToken)
		logrus.Infof("Downloading docker image: %s", downloadUrl)
		closer, err = downloadFile(name, downloadUrl, "/tmp")
		if err != nil {
			return fmt.Errorf("Failed to download remote Docker image tarball file, error: %s", err.Error())
		}
		_, err = im.dockerClient.ImageLoad(context.Background(), closer, false)
		//close the file stream anyway.
		closer.Close()
		if err != nil {
			return fmt.Errorf("Failed to load local tarball docker image file to the Docker daemon, error: %s", err.Error())
		}
		logrus.Infof("Docker image %s had completely loaded into Docker daemon!", v.ImageName)
	}
	return nil
}

func downloadFile(name, url string, dest string) (io.ReadCloser, error) {
	start := time.Now()
	filePath := fmt.Sprintf("%s/%s.tar", dest, name)
	if _, err := os.Stat(filePath); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		out, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}
		defer out.Close()
		headResp, err := http.Head(url)
		if err != nil {
			return nil, err
		}
		defer headResp.Body.Close()
		_, err = strconv.Atoi(headResp.Header.Get("Content-Length"))
		if err != nil {
			return nil, err
		}
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return nil, err
		}
		elapsed := time.Since(start)
		logrus.Debugf("Download %s completed in %s", url, elapsed)
	}
	return os.Open(filePath)
}
