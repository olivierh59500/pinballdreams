// Package sound plays native tracker notes and original PCM samples in Go.
package sound

import (
	"fmt"
	"io/fs"

	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

type Sample = audiodata.Sample
type Note = audiodata.Note
type Command = audiodata.Command
type Module = audiodata.Module

// Decode returns the table's already-extracted, immutable musical data.
func Decode(t *resources.Table) (*Module, error) {
	if t.Audio == nil {
		return nil, fmt.Errorf("extract native audio before loading a table")
	}
	return t.Audio, t.Audio.Validate()
}

func DecodeFrontend(source fs.FS, intro bool) (*Module, error) {
	hash := "c441151fecc503d418068f904fb4c0e86fc47f20644dcd130e565510d7e4317f"
	if intro {
		hash = "3e8f67df8f3d1063c2a8f1a958b42c1bb2d82c3829a3cf0cf5b481b66602e74e"
	}
	return audiodata.LoadFS(source, audiodata.FrontendFilename(intro), hash)
}
