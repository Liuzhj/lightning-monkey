package utils

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)

func Test_GetDockerRepositoryName(t *testing.T) {
	repo := GetDockerRepositoryName("mirrorgooglecontainers/etcd:3.2.24")
	assert.True(t, repo == "docker.io/mirrorgooglecontainers")
}

func Test_GetDockerRepositoryName2(t *testing.T) {
	repo := GetDockerRepositoryName("etcd:3.2.24")
	assert.True(t, repo == "docker.io")
}

func Test_GetDockerRepositoryName3(t *testing.T) {
	repo := GetDockerRepositoryName("gcr.io/mirrorgooglecontainers/etcd:3.2.24")
	assert.True(t, repo == "gcr.io/mirrorgooglecontainers")
}

func Test_GetDockerRepositoryName4(t *testing.T) {
	repo := GetDockerRepositoryName("gcr.io/mirrorgooglecontainers/a/b/c/etcd:3.2.24")
	assert.True(t, repo == "gcr.io/mirrorgooglecontainers/a/b/c")
}
