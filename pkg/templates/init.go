package templates

import "sync"

func init() {
	if lockObj == nil {
		lockObj = &sync.RWMutex{}
	}
	if ts == nil {
		ts = make(map[string]map[string]string)
	}
}
