package highscore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct{ Directory string }

type savedBook struct {
	Version int
	Scores  Book
}

func (s Store) Load(defaults Book, allowed string) (Book, error) {
	data, err := os.ReadFile(filepath.Join(s.Directory, "highscores.json"))
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, err
	}
	var saved savedBook
	if err := json.Unmarshal(data, &saved); err != nil {
		return defaults, fmt.Errorf("saved high scores: %w", err)
	}
	if saved.Version != 1 {
		return defaults, fmt.Errorf("unsupported saved high-score version")
	}
	if err := saved.Scores.Validate(allowed); err != nil {
		return defaults, err
	}
	return saved.Scores, nil
}

// Save publishes complete records and leaves the previous file intact on a
// failed write. The application owns this directory on desktop and Android.
func (s Store) Save(book Book) error {
	data, err := json.MarshalIndent(savedBook{Version: 1, Scores: book}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(s.Directory, ".highscores-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(s.Directory, "highscores.json"))
}
