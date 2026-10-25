package rules

import (
	"slices"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestBallTransitionPreservesPresentationAndOtherPlayerDrawStates(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	r.Art(false, false)
	r.Player = 3
	r.advanceArtwork(false, false)
	state := r.artwork
	other := append([]bool(nil), state.byPlayer[3]...)
	r.NewBall(0)
	if r.artwork != state || !slices.Equal(state.byPlayer[3], other) {
		t.Fatal("new ball discarded the common surface or another player's draw cache")
	}
}

func TestBallLampResetKeepsSharedPhasesAndClearsTemporaryOwnership(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	for id, lamp := range r.Lamps {
		lamp.Flags = lampFlash
		lamp.Players, lamp.Blink, lamp.Phase = 9, true, true
		r.Player = 0
		r.resetBallLamps()
		if lamp.Flags&lampFlash != 0 || lamp.Players != 0 || !lamp.Blink || !lamp.Phase {
			t.Fatalf("lamp %x lost shared state during the ball reset: %+v", id, lamp)
		}
		lamp.Flags = lampPersistent | lampFlash
		lamp.Players = 9
		r.resetBallLamps()
		if lamp.Players != 9 || lamp.Flags != lampPersistent || !lamp.Blink || !lamp.Phase {
			t.Fatal("persistent ownership or shared phases were cleared with temporary state")
		}
		break
	}
}
