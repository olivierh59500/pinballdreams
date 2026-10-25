package game

import "github.com/olivierh59500/pinballdreams/internal/session"

// introTouch waits for a tap's release so a second finger can join it.
// A two-finger hold supplies the opening's secondary control and consumes
// the gesture; lifting those fingers must not also skip the credits.
type introTouch struct{ pending, cancelTap bool }

func (t *introTouch) apply(input *session.Input, contacts []pointerContact, consumed map[pointerID]bool, width, height int) {
	down, inside, fresh := 0, 0, false
	for _, contact := range contacts {
		if contact.id.mouse || consumed[contact.id] {
			continue
		}
		down++
		if contact.x >= 0 && contact.y >= 0 && contact.x < width && contact.y < height {
			inside++
			fresh = fresh || contact.pressed
		}
	}
	if down == 0 {
		if t.pending && !t.cancelTap {
			input.Skip = true
		}
		*t = introTouch{}
		return
	}
	if !t.pending {
		if !fresh {
			return
		}
		t.pending = true
	}
	if inside != down {
		t.cancelTap = true
	}
	if inside >= 2 {
		t.cancelTap = true
		input.IntroExtraCredits = true
	}
}
