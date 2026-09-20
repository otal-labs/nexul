//go:build !embed

package webui

import "io/fs"

// Assets returns nil without the embed tag; the server still boots but serves no static assets.
func Assets() fs.FS {
	return nil
}
