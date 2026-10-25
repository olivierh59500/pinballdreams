package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

// pendingInput retains commands between host polls and a logical game step.
// Held launcher/secondary controls use the latest sample; short flipper and
// nudge taps get one game step. Relative travel and text retain their order.
type pendingInput struct {
	input    session.Input
	keys     []ebiten.Key
	initials []session.InitialsAction
}

func (p *pendingInput) add(next session.Input, pressed []ebiten.Key, initials ...session.InitialsAction) {
	old := p.input
	if next.InitialsEdit != session.InitialsNone {
		p.initials = append(p.initials, next.InitialsEdit)
	}
	p.initials = append(p.initials, initials...)
	next.InitialsEdit = session.InitialsNone
	next.Left = next.Left || old.Left
	next.Right = next.Right || old.Right
	next.Nudge = next.Nudge || old.Nudge
	next.Start = next.Start || old.Start
	next.Pause = next.Pause || old.Pause
	next.Back = next.Back || old.Back
	next.Music = next.Music || old.Music
	next.Skip = next.Skip || old.Skip
	next.StartPressed = next.StartPressed || old.StartPressed
	next.MusicPressed = next.MusicPressed || old.MusicPressed
	next.PauseKeyPressed = next.PauseKeyPressed || old.PauseKeyPressed
	next.ExitTable = next.ExitTable || old.ExitTable
	next.SpringMotion += old.SpringMotion
	next.SpringClick = next.SpringClick || old.SpringClick
	next.HighScores = next.HighScores || old.HighScores
	next.Backspace = next.Backspace || old.Backspace
	next.QuitConfirm = next.QuitConfirm || old.QuitConfirm
	next.QuitCancel = next.QuitCancel || old.QuitCancel
	next.AnyKey = next.AnyKey || old.AnyKey
	next.Text = old.Text + next.Text
	if next.Table == 0 {
		next.Table = old.Table
	}
	if next.Players == 0 {
		next.Players = old.Players
	}
	if next.MenuKey == 0 {
		next.MenuKey = old.MenuKey
	}
	p.input = next
	p.keys = append(p.keys, pressed...)
}

func (p *pendingInput) reset() {
	p.finishFrame()
	p.initials = p.initials[:0]
}

func (p *pendingInput) finishFrame() {
	p.input = session.Input{}
	p.keys = p.keys[:0]
}

// take preserves one editing command per logical prompt update. Later taps
// remain queued rather than replacing an earlier confirmation or character.
func (p *pendingInput) take() session.Input {
	input := p.input
	if len(p.initials) != 0 {
		input.InitialsEdit = p.initials[0]
		p.initials = p.initials[1:]
	}
	return input
}
