package templates

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)

func Test_ParseSimpleVersion(t *testing.T) {
	v, _ := parseVersions("1.12.3")
	assert.True(t, v != nil)
	assert.True(t, len(v) == 3)
	assert.True(t, v[0] == "1")
	assert.True(t, v[1] == "1.12")
	assert.True(t, v[2] == "1.12.3")
}

func Test_ParseSimpleVersion2(t *testing.T) {
	v, _ := parseVersions("1.12")
	assert.True(t, v != nil)
	assert.True(t, len(v) == 2)
	assert.True(t, v[0] == "1")
	assert.True(t, v[1] == "1.12")
}

func Test_ParseSimpleVersion3(t *testing.T) {
	v, _ := parseVersions("1")
	assert.True(t, v != nil)
	assert.True(t, len(v) == 1)
	assert.True(t, v[0] == "1")
}

func Test_ParseIllegalVersion1(t *testing.T) {
	v, err := parseVersions(".1")
	assert.NotNil(t, err)
	assert.Nil(t, v)
}

func Test_ParseIllegalVersion2(t *testing.T) {
	v, _ := parseVersions("1.12.3.")
	t.Logf("%#v", v)
	assert.True(t, v != nil)
	assert.True(t, len(v) == 3)
	assert.True(t, v[0] == "1")
	assert.True(t, v[1] == "1.12")
	assert.True(t, v[2] == "1.12.3")
}

func Test_ParseIllegalVersion3(t *testing.T) {
	v, _ := parseVersions("1.12.*...")
	t.Logf("%#v", v)
	assert.True(t, v != nil)
	assert.True(t, len(v) == 3)
	assert.True(t, v[0] == "1")
	assert.True(t, v[1] == "1.12")
	assert.True(t, v[2] == "1.12.*")
}

func Test_SuccessfulGetTemplate(t *testing.T) {
	role := "X"
	SetTemplate(role, []string{"1.12", "1.13"}, "abc")
	v, err := GetTemplate(role, "1.13")
	assert.Nil(t, err)
	assert.True(t, v == "abc")
}

func Test_SuccessfulGetTemplate2(t *testing.T) {
	role := "X"
	SetTemplate(role, []string{"1.12", "1.13"}, "abc")
	v, err := GetTemplate(role, "1.13.1")
	assert.Nil(t, err)
	assert.True(t, v == "abc")
}

func Test_SuccessfulGetTemplate3(t *testing.T) {
	role := "X"
	SetTemplate(role, []string{"1.12", "1.13"}, "abc")
	SetTemplate(role, []string{"1.13.1"}, "xyz")
	v, err := GetTemplate(role, "1.13.1")
	assert.Nil(t, err)
	assert.True(t, v == "xyz")
}

func Test_SuccessfulGetTemplate4(t *testing.T) {
	role := "X"
	SetTemplate(role, []string{"1.12", "1.13"}, "abc")
	SetTemplate(role, []string{"1.13.1"}, "xyz")
	v, err := GetTemplate(role, "1.13.*")
	assert.Nil(t, err)
	assert.True(t, v == "abc")
}

func Test_SuccessfulGetTemplate5(t *testing.T) {
	role := "X"
	SetTemplate(role, []string{"*"}, "ppp")
	SetTemplate(role, []string{"1.12", "1.13"}, "abc")
	SetTemplate(role, []string{"1.13.1"}, "xyz")
	v, err := GetTemplate(role, "1.1313.*")
	assert.Nil(t, err)
	assert.True(t, v == "ppp")
}

func Test_SuccessfulGetTemplate6(t *testing.T) {
	role := "X"
	SetTemplate(role, []string{"*"}, "ppp")
	v, err := GetTemplate(role, "1.1313.*")
	assert.Nil(t, err)
	assert.True(t, v == "ppp")
}

func Test_FailedGetTemplate(t *testing.T) {
	role := "X"
	SetTemplate(role, []string{"*"}, "ppp")
	v, err := GetTemplate(role, ".1.1313.*")
	assert.NotNil(t, err)
	assert.True(t, v == "")
	t.Logf("%#v", err)
}

func Test_FailedGetTemplate2(t *testing.T) {
	role := "Y"
	SetTemplate("X", []string{"*"}, "ppp")
	v, err := GetTemplate(role, "1.1313.*")
	assert.Nil(t, err)
	assert.True(t, v == "")
}
