package audiodata

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

func TableFilename(id int) string { return fmt.Sprintf("table%d-audio.json", id+1) }

func FrontendFilename(intro bool) string {
	if intro {
		return "intro-audio.json"
	}
	return "menu-audio.json"
}

func LoadFS(source fs.FS, name, expectedSHA256 string) (*Module, error) {
	data, err := fs.ReadFile(source, name)
	if err != nil {
		return nil, fmt.Errorf("extract native audio assets first: %w", err)
	}
	var m Module
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("native audio assets: %w", err)
	}
	if expectedSHA256 != "" && m.SourceSHA256 != expectedSHA256 {
		return nil, fmt.Errorf("native audio does not match its source record")
	}
	return &m, m.Validate()
}

func (m *Module) Validate() error {
	if m.Version != 2 || len(m.SourceSHA256) != 64 || len(m.Samples) < 1 || len(m.Samples) > 31 || len(m.Patterns) == 0 || len(m.Orders) == 0 || m.InitialOrder < 0 || m.InitialOrder >= len(m.Orders) {
		return fmt.Errorf("invalid native audio schema")
	}
	if len(m.BallStart) > 0 {
		spring := m.SpringRelease
		if spring.Cue.Action != Sound || spring.Cue.Note.Sample < 1 || spring.Cue.Note.Sample > len(m.Samples) || spring.Cue.Note.Pitch < 1 || spring.Cue.Note.Pitch > 36 || spring.Cue.Note.Effect != 12 || spring.VolumeStep < 1 || spring.VolumeLimit < 1 || spring.VolumeLimit > 64 {
			return fmt.Errorf("invalid native spring release cue")
		}
	}
	for _, sample := range m.Samples {
		if sample.Offset < 0 || sample.Length < 0 || sample.Offset > len(m.PCM)-sample.Length || sample.Volume < 0 || sample.Volume > 64 || sample.Finetune < 0 || sample.Finetune > 15 || sample.LoopStart < 0 || sample.LoopLength < 0 {
			return fmt.Errorf("invalid native sample range")
		}
	}
	for _, pattern := range m.Orders {
		if pattern < 0 || pattern < 128 && pattern >= len(m.Patterns) {
			return fmt.Errorf("invalid native pattern selection")
		}
	}
	for _, pattern := range m.Patterns {
		for _, row := range pattern {
			for _, note := range row {
				if note.Sample < 0 || note.Sample > len(m.Samples) || note.Pitch < 0 || note.Pitch > 36 || note.Effect < 0 || note.Effect > 15 || note.Parameter < 0 || note.Parameter > 255 {
					return fmt.Errorf("invalid native tracker note")
				}
			}
		}
	}
	return nil
}
