package main

import (
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func createCgroupsSubDirectories() {
	logrus.Infof("Creating cgroups sub-directories...")
	rootPaths := []string{
		"/sys/fs/cgroup/blkio",
		"/sys/fs/cgroup/cpu",
		"/sys/fs/cgroup/cpuacct",
		"/sys/fs/cgroup/cpuset",
		"/sys/fs/cgroup/devices",
		"/sys/fs/cgroup/freezer",
		"/sys/fs/cgroup/hugetlb",
		"/sys/fs/cgroup/memory",
		"/sys/fs/cgroup/net_cls",
		"/sys/fs/cgroup/net_prio",
		"/sys/fs/cgroup/perf_event",
		"/sys/fs/cgroup/pids",
		"/sys/fs/cgroup/systemd",
	}
	subDirectories := []string{"kube-reserved", "system-reserved"}
	var path string
	for _, v := range rootPaths {
		for _, subPath := range subDirectories {
			path = filepath.Join(v, subPath)
			_ = os.MkdirAll(path, 0755) //rwxr-xr-x
		}
	}
}
