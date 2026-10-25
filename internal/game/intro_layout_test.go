package game

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/internal/session"
)

func TestOpeningLayoutAndPointerUseTheFullDeclaredWidth(t *testing.T) {
	g, err := New(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	width, height := g.Layout(960, 768)
	if width != 672 || height != 512 {
		t.Fatalf("opening layout is %d×%d", width, height)
	}
	var input session.Input
	g.Session.ApplyPointer(&input, 660, 100, height)
	if !input.Skip {
		t.Fatal("opening's right margin did not accept its skip gesture")
	}
	width, height = g.Layout(600, 1000)
	if width != 672 || height != 1084 {
		t.Fatalf("portrait opening layout is %d×%d", width, height)
	}
}
