package main

import (
	"embed"
	"io/fs"
)

//go:embed frontend/dist
var embeddedFrontend embed.FS

func frontendAssets() fs.FS {
	assets, err := fs.Sub(embeddedFrontend, "frontend/dist")
	if err != nil {
		return embeddedFrontend
	}
	return assets
}
