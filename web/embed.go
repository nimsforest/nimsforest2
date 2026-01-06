//go:build !dev

// Package web provides the embedded web frontend assets.
package web

import (
	"embed"
	"io/fs"
)

// Assets contains the built web frontend from the out/ directory.
// Build with: cd web && npm run build
//
//go:embed all:out
var assets embed.FS

// GetAssets returns the embedded web assets as an fs.FS.
// The returned filesystem has the out/ prefix stripped.
func GetAssets() (fs.FS, error) {
	return fs.Sub(assets, "out")
}
