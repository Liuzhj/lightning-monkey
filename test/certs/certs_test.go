package certs

import (
	"github.com/g0194776/lightningmonkey/pkg/certs"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_GernerateCerts(t *testing.T) {
	resources, err := certs.GenerateMasterCertificates("/tmp/123123", "0.0.0.0", "192.168.0.0/12")
	require.True(t, err == nil)
	require.True(t, resources != nil)
	require.True(t, resources.GetResources() != nil)
	require.True(t, len(resources.GetResources()) > 0)
}
