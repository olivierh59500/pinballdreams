package assets_test

import (
	"io/fs"
	"path"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
)

func TestApplicationBundleContainsOnlyNativeAssets(t *testing.T) {
	count := 0
	err := fs.WalkDir(assets.Files(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			count++
			if path.Ext(name) != ".json" {
				t.Fatalf("application embeds an original record or binary: %s", name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 15 {
		t.Fatalf("native application bundle has %d files; want 15", count)
	}
}
