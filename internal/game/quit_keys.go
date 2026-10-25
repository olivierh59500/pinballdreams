package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

// quitKeys retains only releases consumed by a host-pause resume. The original
// ordinary question polls key releases; a question inside manual pause polls
// fresh presses through its separate pause routine.
type quitKeys struct{ consumed map[ebiten.Key]bool }

func (q *quitKeys) apply(input *session.Input, asking, paused bool, pressed, released []ebiten.Key) {
	for _, key := range released {
		consumed := q.consumed[key]
		delete(q.consumed, key)
		if asking && !paused && !consumed && key != ebiten.KeyEscape {
			input.AnyKey = true
			input.QuitConfirm = input.QuitConfirm || key == ebiten.KeyY
		}
	}
	if asking && paused {
		for _, key := range pressed {
			if key != ebiten.KeyEscape {
				input.AnyKey = true
				input.QuitConfirm = input.QuitConfirm || key == ebiten.KeyY
			}
		}
	}
}

func (q *quitKeys) consumeResume(pressed []ebiten.Key) {
	if q.consumed == nil {
		q.consumed = make(map[ebiten.Key]bool)
	}
	for _, key := range pressed {
		q.consumed[key] = true
	}
}
