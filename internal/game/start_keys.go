package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

var playerKeys = [...]ebiten.Key{ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5, ebiten.KeyF6, ebiten.KeyF7, ebiten.KeyF8}

// applyStartKeys gives a new command priority over other held keys. Simultaneous
// fresh function keys retain the adapter's deterministic highest-key priority.
func applyStartKeys(s *session.Session, input *session.Input, down, pressed func(ebiten.Key) bool) {
	enterPressed := pressed(ebiten.KeyEnter) || pressed(ebiten.KeyNumpadEnter)
	if enterPressed {
		input.Start, input.StartPressed = true, true
		input.Players = 0
	}
	if s.NameEntry != nil {
		return
	}
	heldChoice, freshChoice := 0, 0
	for n, key := range playerKeys {
		if down(key) {
			heldChoice = n + 1
		}
		if pressed(key) {
			freshChoice = n + 1
		}
	}
	if s.Table == nil {
		input.Table = heldChoice
		if freshChoice != 0 {
			input.Table = freshChoice
		}
		return
	}
	input.Start = input.Start || heldChoice != 0
	if freshChoice != 0 {
		input.Start, input.StartPressed, input.Players = true, true, freshChoice
	}
}
