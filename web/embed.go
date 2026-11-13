package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html css/* js/*
var content embed.FS

// GetFS returns the filesystem with root files accessible.
func GetFS() fs.FS {
	return content
}
