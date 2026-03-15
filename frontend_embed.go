package main

import (
	"embed"
	"io/fs"
)

//go:embed web/dist
var embeddedFrontendDist embed.FS

func frontendFS() (fs.FS, error) {
	return fs.Sub(embeddedFrontendDist, "web/dist")
}
