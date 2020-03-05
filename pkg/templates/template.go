package templates

import (
	"fmt"
	"strings"
	"sync"
)

var (
	lockObj *sync.RWMutex
	ts      map[string] /*role name*/ map[string]string /*version -> template*/
)

const (
	DEFAULT_VERSION = "*"
)

func GetTemplate(role string, version string) (string, error) {
	lockObj.RLock()
	defer lockObj.RUnlock()
	var isOK bool
	var tempValue string
	var tempMap map[string]string
	if tempMap, isOK = ts[role]; !isOK {
		return "", nil
	}
	//try to parse all of available composition by given version.
	//example: "1.12.3" -> "1", "1.12", "1.12.3"
	vs, err := parseVersions(version)
	if err != nil {
		return "", err
	}
	if vs == nil || len(vs) == 0 {
		return "", nil
	}
	for i := len(vs) - 1; i >= 0; i-- {
		if tempValue, isOK = tempMap[vs[i]]; isOK {
			return tempValue, nil
		}
	}
	if tempValue, isOK = tempMap[DEFAULT_VERSION]; isOK {
		return tempValue, nil
	}
	return "", nil
}

func SetTemplate(role string, versions []string, templateValue string) {
	lockObj.Lock()
	defer lockObj.Unlock()
	var isOK bool
	var tempMap map[string]string
	if tempMap, isOK = ts[role]; !isOK {
		tempMap = make(map[string]string)
	}
	for i := 0; i < len(versions); i++ {
		tempMap[versions[i]] = templateValue
	}
	ts[role] = tempMap
}

func parseVersions(version string) ([]string, error) {
	var result []string
	if version == "" {
		return result, nil
	}
	if strings.Index(version, ".") == 0 {
		return nil, fmt.Errorf("version value CANNOT start with punctuation")
	}
	offset := 0
	for i := 0; i < len(version); i++ {
		if version[i] == '.' {
			continue
		}
		offset = strings.Index(version[i:], ".")
		if offset != -1 {
			i += offset
			result = append(result, version[:i])
		} else if i+1 == len(version) {
			result = append(result, version[:i+1])
		}
	}
	return result, nil
}
