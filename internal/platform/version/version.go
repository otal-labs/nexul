// Package version holds the single build-stamped version both binaries read (server and runner).
package version

import "strings"

// Version is the release tag stamped at build time, e.g. -ldflags "-X .../version.Version=v0.2.1-beta.3".
// "dev" means a local, unstamped build.
var Version = "dev"

// Channel reports "dev" for an unstamped build, "beta" for any prerelease (a dash after the semver, e.g.
// v0.2.1-beta.3), else "stable". release.yml applies the same rule when it picks tags.
func Channel() string {
	if Version == "dev" {
		return "dev"
	}
	if strings.Contains(Version, "-") {
		return "beta"
	}
	return "stable"
}

// IsRelease reports whether Version was stamped at build time.
func IsRelease() bool {
	return Version != "dev"
}
