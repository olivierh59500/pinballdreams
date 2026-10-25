package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

type pointerID struct {
	touch ebiten.TouchID
	mouse bool
}

type pointerContact struct {
	id          pointerID
	x, y        int
	pressed     bool
	hostControl bool
}

// inputState keeps one gesture from acting on both sides of a modal transition.
// Its identities belong to current host contacts, never to game resources.
type inputState struct {
	waitForRelease bool
	canvasHeight   int
	consumed       map[pointerID]bool
	cancelled      map[pointerID]bool
	flippers       map[pointerID]flipperHold
	spring         map[pointerID]bool
	introTouch     introTouch
}

type flipperHold struct{ left, right bool }

// CancelTouch consumes a platform-cancelled contact until its removal reaches
// the input snapshot. Foreground navigation and other fingers remain usable.
func (g *Game) CancelTouch(id ebiten.TouchID) {
	pointer := pointerID{touch: id}
	if g.controls.cancelled == nil {
		g.controls.cancelled = make(map[pointerID]bool)
	}
	g.controls.cancelled[pointer] = true
	g.controls.consume(pointer)
	delete(g.controls.flippers, pointer)
	delete(g.controls.spring, pointer)
	g.Session.CancelLaunch()
	g.pending.input.Launch = false
	g.pending.input.SpringMotion, g.pending.input.SpringClick = 0, false
}

// menuLetter reports physical letter presses for the supplied hidden ticker
// sequences. Other pressed keys cancel a partial sequence in the same way.
func menuLetter(key ebiten.Key) byte {
	if key >= ebiten.KeyA && key <= ebiten.KeyZ {
		return 'A' + byte(key-ebiten.KeyA)
	}
	return session.OtherMenuKey
}

func continuingContact(id pointerID, contacts []pointerContact) bool {
	for _, contact := range contacts {
		if contact.id == id && !contact.pressed {
			return true
		}
	}
	return false
}

func (p *inputState) clearCancelled(contacts []pointerContact) {
	for id := range p.cancelled {
		present := false
		for _, contact := range contacts {
			present = present || contact.id == id
		}
		if !present {
			delete(p.cancelled, id)
		}
	}
}

func (p *inputState) apply(s *session.Session, keys session.Input, contacts []pointerContact, keyDown bool, height int) session.Input {
	p.clearCancelled(contacts)
	for id := range p.consumed {
		if !continuingContact(id, contacts) {
			delete(p.consumed, id)
		}
	}
	for id := range p.flippers {
		if !continuingContact(id, contacts) {
			delete(p.flippers, id)
		}
	}
	for id := range p.spring {
		if !continuingContact(id, contacts) {
			delete(p.spring, id)
		}
	}
	previousHeight := p.canvasHeight
	p.canvasHeight = height
	if previousHeight != 0 && previousHeight != height && s.Table != nil {
		for _, contact := range contacts {
			if !contact.id.mouse && !contact.pressed && !p.consumed[contact.id] {
				// Rotation can remap a held finger before the platform's
				// cancellation arrives. Freeze before interpreting that movement
				// as a spring release, nudge or a different flipper grip.
				s.Suspend()
				p.waitForRelease = true
				p.flippers = nil
				break
			}
		}
	}
	if p.waitForRelease {
		p.introTouch = introTouch{}
		p.spring = nil
		if !keyDown && len(contacts) == 0 {
			p.waitForRelease = false
		}
		// The release boundary itself remains neutral. Android may deliver
		// a delayed Back event or text together with the released contacts.
		return session.Input{}
	}
	if s.Intro == nil || s.Paused || keys.Pause || keys.PauseKeyPressed {
		p.introTouch = introTouch{}
	} else {
		p.introTouch.apply(&keys, contacts, p.consumed, s.Intro.Size().X, height)
	}
	if s.NameEntry != nil {
		if s.Paused {
			// Resume cannot type P, finish the entry or edit a character.
			keys = session.Input{Pause: keys.Pause, PauseKeyPressed: keys.PauseKeyPressed, AnyKey: keys.AnyKey}
		} else {
			// P belongs to the supplied initials alphabet during active editing.
			keys.Pause = false
			keys.PauseKeyPressed = false
		}
	}
	pausing := keys.Pause || keys.PauseKeyPressed && !s.DefersKeyboardPause()
	modal := s.Paused || s.AskingToQuit() || s.NameEntry != nil || !s.Playing && !s.PreparingBall()
	holdFlippers := s.Playing && s.NameEntry == nil && !s.AskingToQuit()
	for _, contact := range contacts {
		if contact.hostControl || p.consumed[contact.id] || p.cancelled[contact.id] {
			continue
		}
		if s.Intro != nil && !s.Paused && !pausing && !contact.id.mouse {
			// Opening touches form a tap or a two-finger secondary hold.
			// Mouse clicks retain their immediate primary-button action.
			continue
		}
		if modal && !contact.pressed {
			if s.Paused && holdFlippers {
				hold := p.flippers[contact.id]
				keys.Left, keys.Right = keys.Left || hold.left, keys.Right || hold.right
			}
			continue
		}
		if !modal && p.spring[contact.id] {
			// A launcher gesture belongs to its finger until release. A
			// changed coordinate mapping is not a physical finger lift.
			keys.Launch = true
			continue
		}
		var action session.Input
		s.ApplyPointer(&action, contact.x, contact.y, height)
		if contact.pressed && action.Start {
			keys.StartPressed = true
		}
		if contact.pressed && action.Music {
			keys.MusicPressed = true
		}
		if !modal {
			if action.Launch && !contact.id.mouse {
				if p.spring == nil {
					p.spring = make(map[pointerID]bool)
				}
				p.spring[contact.id] = true
			}
			// Remember this contact's flipper contribution independently of
			// keyboard buttons and the other contacts already accumulated.
			if action.Left || action.Right {
				if p.flippers == nil {
					p.flippers = make(map[pointerID]flipperHold)
				}
				p.flippers[contact.id] = flipperHold{left: action.Left, right: action.Right}
			} else {
				delete(p.flippers, contact.id)
			}
		}
		s.ApplyPointer(&keys, contact.x, contact.y, height)
		if modal {
			p.consume(contact.id)
		}
	}
	if pausing {
		// Keep an existing flipper grip so a paused cradle does not drop
		// on resume. Other gestures cannot carry into the resumed table.
		for _, contact := range contacts {
			if hold := p.flippers[contact.id]; holdFlippers && (hold.left || hold.right) {
				continue
			}
			p.consume(contact.id)
		}
	}
	return keys
}

func (p *inputState) consume(id pointerID) {
	if p.consumed == nil {
		p.consumed = make(map[pointerID]bool)
	}
	p.consumed[id] = true
}
