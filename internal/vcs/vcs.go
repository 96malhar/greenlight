package vcs

import (
	"runtime/debug"
)

func Version() string {
	// Use debug.ReadBuildInfo() to retrieve a debug.BuildInfo struct. If this available,
	// the ok value will be true, and we return the pseudo-version contained in the
	// Main.Version field.
	bi, ok := debug.ReadBuildInfo()
	if ok {
		return bi.Main.Version
	}
	return ""
}
