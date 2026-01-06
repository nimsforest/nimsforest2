// Package webview provides an HTTP server for the isometric webview visualization.
package webview

import (
	"embed"
	"io/fs"
)

// webFiles embeds the built web frontend from web/out.
// This allows the frontend to be distributed with the binary.
//
//go:embed all:dist
var webFiles embed.FS

// GetEmbeddedWebFS returns the embedded web files as an fs.FS.
// The returned filesystem has the dist/ prefix stripped so files
// can be accessed directly (e.g., "index.html" instead of "dist/index.html").
func GetEmbeddedWebFS() (fs.FS, error) {
	return fs.Sub(webFiles, "dist")
}
