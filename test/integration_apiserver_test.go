package test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/g0194776/lightningmonkey/pkg/entities"
	"github.com/googleapis/gnostic/compiler"
	assert "github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	once                sync.Once
	dc                  *client.Client
	depenedDockerImages = []string{"docker.io/bitnami/etcd:latest"}
)

func initDockerClient() {
	c, err := client.NewEnvClient()
	if err != nil {
		panic(fmt.Errorf("Failed to initialize docker client, error: %s", err.Error()))
	}
	dc = c
}

func pullDependedImage() {
	for i := 0; i < len(depenedDockerImages); i++ {
		fmt.Printf("# Pulling docker image: %s\n", depenedDockerImages[i])
		reader, err := dc.ImagePull(context.Background(), depenedDockerImages[i], types.ImagePullOptions{})
		if err != nil {
			panic(fmt.Errorf("Failed to pull docker image, error: %s", err.Error()))
		}
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			panic(err)
		}
	}
}

func tarSourceCode() (string, error) {
	src := "/Users/kevinyang/Documents/golang/src/github.com/g0194776/lightningmonkey"
	dstFileStr := filepath.Join(src, "../", "1.tar")
	os.RemoveAll(dstFileStr)
	file, err := os.Create(dstFileStr)
	if err != nil {
		return "", err
	}
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return dstFileStr, fmt.Errorf("Unable to tar files - %v", err.Error())
	}
	mw := file
	gzw := gzip.NewWriter(mw)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()
	// walk path
	err = filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		// return on any error
		if err != nil {
			return err
		}
		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}
		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))
		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}
		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}
		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()
		return nil
	})
	return dstFileStr, err
}

func buildAPIServer(imageName, imageTag string) {
	dockerBuildContext, err := os.Open("/Users/kevinyang/Documents/golang/src/github.com/g0194776/1.tar")
	defer dockerBuildContext.Close()

	buildOptions := types.ImageBuildOptions{
		SuppressOutput: true,
		PullParent:     true,
		Tags:           []string{fmt.Sprintf("%s:%s", imageName, imageTag)},
		Context:        dockerBuildContext,
		Dockerfile:     "Dockerfile.apiserver",
	}
	imageBuildResponse, err := dc.ImageBuild(context.Background(), dockerBuildContext, buildOptions)
	if err != nil {
		panic(err)
	}
	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		panic(err)
	}
}

func removeImage(imageName, imageTag string) {
	images, err := dc.ImageList(context.Background(), types.ImageListOptions{All: true})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", images)
	//for _, v := range images {
	//	v.
	//}
}

func startDependedETCD() string {
	fmt.Println("Start depended ETCD image...")
	resp, err := dc.ContainerCreate(context.Background(), &container.Config{
		Image:   "bitnami/etcd:latest",
		Tty:     false,
		Volumes: map[string]struct{}{},
		Env:     []string{"ALLOW_NONE_AUTHENTICATION=yes"},
	}, &container.HostConfig{
		Privileged:    true,
		NetworkMode:   "bridge",
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}, &network.NetworkingConfig{}, "lm-etcd")
	if err != nil {
		panic(fmt.Sprintf("Failed to create container, error: %s", err.Error()))
	}
	if err = dc.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(fmt.Sprintf("Failed to start container, error: %s", err.Error()))
	}
	out, err := dc.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(fmt.Sprintf("Failed to retrieve container logs, error: %s", err.Error()))
	}
	_, _ = io.Copy(os.Stdout, out)
	fmt.Printf("Using depended ETCD container ID: %s\n", resp.ID)
	time.Sleep(time.Second * 5)
	return resp.ID
}

func removeContainers(names ...string) {
	containers, err := dc.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}
	if containers == nil || len(containers) == 0 {
		return
	}
	for i := 0; i < len(containers); i++ {
		c := containers[i]
		if c.Names == nil || len(c.Names) == 0 {
			continue
		}
		for j := 0; j < len(c.Names); j++ {
			if compiler.StringArrayContainsValue(names, c.Names[j]) {
				err = dc.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func startAPIServer(imageName, imageTag string) {
	fmt.Println("Cleaning running containers...")
	removeContainers("/lm-etcd", "/lm-apiserver")
	cId := startDependedETCD()
	resp, err := dc.ContainerCreate(context.Background(), &container.Config{
		Image:   fmt.Sprintf("%s:%s", imageName, imageTag),
		Tty:     false,
		Env:     []string{"BACKEND_STORAGE_ARGS=ENDPOINTS=http://127.0.0.1:2379"},
		Volumes: map[string]struct{}{},
	}, &container.HostConfig{
		Privileged:    true,
		NetworkMode:   container.NetworkMode("container:" + cId),
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{
				nat.PortBinding{HostIP: "0.0.0.0", HostPort: "8080"},
			},
		},
	}, &network.NetworkingConfig{}, "lm-apiserver")
	if err != nil {
		panic(fmt.Sprintf("Failed to create container, error: %s", err.Error()))
	}
	if err = dc.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(fmt.Sprintf("Failed to start container, error: %s", err.Error()))
	}
	out, err := dc.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(fmt.Sprintf("Failed to retrieve container logs, error: %s", err.Error()))
	}
	_, _ = io.Copy(os.Stdout, out)
}

func setup(t *testing.T) func(t *testing.T) {
	var destFile string
	imageName := "g0194776/lightning-monkey/apiserver"
	imageTag := time.Now().Format("150405")
	once.Do(func() {
		t.Log("Setup method call...")
		t.Log("Initializing docker client...")
		initDockerClient()
		//t.Log("Try pulling depended docker images...")
		//pullDependedImage()
		t.Log("Zipping source code...")
		dstFile, err := tarSourceCode()
		if err != nil {
			panic(err)
		}
		destFile = dstFile
		t.Log("Try building apiserver docker image...")
		buildAPIServer(imageName, imageTag)
		startAPIServer(imageName, imageTag)
	})
	return func(t *testing.T) {
		t.Log("Tear down...")
		if destFile != "" {
			os.RemoveAll(destFile)
		}
		//removeContainers("/lm-etcd", "/lm-apiserver")
		//removeImage(imageName, imageTag)
	}
}

func Test_NewCluster(t *testing.T) {
	tm := setup(t)
	defer tm(t)
	clusterSettings := entities.LightningMonkeyClusterSettings{}
	data, err := json.Marshal(&clusterSettings)
	if err != nil {
		t.Fatal(err)
	}
	reader := bytes.NewReader(data)
	req, err := http.NewRequest("POST", "http://127.0.0.1:8080", reader)
	if err != nil {
		t.Fatal(err)
	}
	client := http.Client{Timeout: time.Second * 30, Transport: http.DefaultTransport}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(respData))
	assert.True(t, resp.StatusCode == http.StatusOK)
	respObj := entities.CreateClusterResponse{}
	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, respObj.ClusterId != "")
	assert.True(t, respObj.ErrorId == 0)
	assert.True(t, respObj.Reason == "")
	//check ETCD details.
}
