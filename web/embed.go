package web

import (
	"embed"
	"io/fs"
)

// DistFS embeds the frontend static files.
//
//go:embed dist/*
var dist embed.FS

// GetDistFS returns the embedded filesystem rooted at "dist".
func GetDistFS() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
