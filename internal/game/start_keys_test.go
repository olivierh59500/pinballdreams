package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

func TestFreshPlayerCommandsSurviveOtherHeldKeysAndTouches(t *testing.T) {
	g := readyTouchGame(t, 0)
	held := map[ebiten.Key]bool{}
	apply := func(pressed ebiten.Key, contacts []pointerContact, want int) {
		t.Helper()
		input := session.Input{Start: held[ebiten.KeyEnter]}
		applyStartKeys(g.Session, &input, func(key ebiten.Key) bool { return held[key] }, func(key ebiten.Key) bool { return key == pressed })
		updateContacts(t, g, input, contacts, len(held) != 0)
		if g.Session.PlayerCount != want {
			t.Fatalf("fresh key %v with held keys %v: got %d players, want %d", pressed, held, g.Session.PlayerCount, want)
		}
	}
	held[ebiten.KeyF8] = true
	apply(ebiten.KeyF8, nil, 8)
	held[ebiten.KeyF2] = true
	apply(ebiten.KeyF2, nil, 2)
	held[ebiten.KeyEnter] = true
	apply(ebiten.KeyEnter, nil, 3)
	apply(ebiten.KeyMax+1, nil, 3)
	held[ebiten.KeyF1] = true
	apply(ebiten.KeyF1, nil, 1)
	contact := touch(1, 8, 10, true)
	apply(ebiten.KeyMax+1, []pointerContact{contact}, 2)
	contact.pressed = false
	for range 8 {
		apply(ebiten.KeyMax+1, []pointerContact{contact}, 2)
	}
	apply(ebiten.KeyMax+1, []pointerContact{contact, touch(2, 8, 10, true)}, 3)
}

func TestFreshTableChoiceOverridesAnotherHeldFunctionKey(t *testing.T) {
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	for !s.Menu.Ready() {
		if err := s.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	var input session.Input
	applyStartKeys(s, &input,
		func(key ebiten.Key) bool { return key == ebiten.KeyF1 || key == ebiten.KeyF4 },
		func(key ebiten.Key) bool { return key == ebiten.KeyF1 })
	if err := s.Update(input); err != nil {
		t.Fatal(err)
	}
	for frame := 0; s.Table == nil && frame < 100; frame++ {
		if err := s.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Table == nil || s.Table.ID != 0 {
		t.Fatal("the held F4 replaced the newly pressed Ignition choice")
	}
}
