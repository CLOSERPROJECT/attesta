package main

import (
	"runtime/debug"
	"strings"
)

const versionMajor = "1"

// buildCommit can be set at build time with -ldflags "-X main.buildCommit=<commit>".
var buildCommit string

func applicationVersion() string {
	commit := strings.TrimSpace(buildCommit)
	if commit == "" {
		commit = revisionFromBuildInfo()
	}
	if commit == "" {
		commit = "dev"
	}
	if len(commit) > 12 {
		commit = commit[:12]
	}
	return versionMajor + "." + commit
}

func revisionFromBuildInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}
