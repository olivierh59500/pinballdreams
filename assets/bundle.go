// Package assets embeds the native game resources included with the project.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed generated/presentation.json generated/table*-rules.json generated/table*-art-physics.json generated/*-audio.json
var bundle embed.FS

func Files() fs.FS {
	files, err := fs.Sub(bundle, "generated")
	if err != nil {
		panic(err)
	}
	return files
}
