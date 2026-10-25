package highscore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistentScoresRoundTripAndRejectCorruption(t *testing.T) {
	store := Store{Directory: t.TempDir()}
	defaults := Book{}
	for table := range defaults {
		for rank := range defaults[table] {
			defaults[table][rank] = Entry{"TSL", uint64(4-rank) * 1000}
		}
	}
	book, err := store.Load(defaults, " TSLABC")
	if err != nil || book != defaults {
		t.Fatal("missing save did not retain supplied defaults", err)
	}
	book[1][0] = Entry{"ABC", 99999}
	if err := store.Save(book); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(defaults, " TSLABC")
	if err != nil || loaded != book {
		t.Fatal("score save did not survive reload", err)
	}
	entries, err := os.ReadDir(store.Directory)
	if err != nil || len(entries) != 1 {
		t.Fatal("temporary score output was not cleaned up", err)
	}
	if err := os.WriteFile(filepath.Join(store.Directory, "highscores.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(defaults, " TSLABC"); err == nil {
		t.Fatal("corrupted high scores were accepted")
	}
}
