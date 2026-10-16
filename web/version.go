package web

import "runtime/debug"

// Version is the build version shown in the footer and at /healthz. Set it at
// build time with:
//
//	-ldflags "-X github.com/dan1165/hyperblur/web.Version=<version>"
//
// When left as "dev" it falls back to the VCS revision reported by the Go build
// info.
var Version = "dev"

func init() {
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	revision := ""
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if modified {
		revision += "-dirty"
	}
	Version = revision
}
