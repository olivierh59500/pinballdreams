package session

import "image/color"

type quitPhase int

const (
	quitQuestion quitPhase = iota
	quitExitStart
	quitAudioFade
	quitPaletteFade
	quitComplete
	quitAborted
)

type quitOutcome int

const (
	quitPending quitOutcome = iota
	quitCancelled
	quitExited
)

type quitFlow struct {
	phase        quitPhase
	paletteSteps int
}

// AskingToQuit selects the supplied confirmation cells for native keyboard
// and touch controls. Idle input and another Escape leave the question open.
func (s *Session) AskingToQuit() bool {
	return s.tablePause != nil && !s.tablePause.suspended && s.tablePause.question || s.quit != nil && s.quit.phase == quitQuestion
}

func (s *Session) beginQuit() {
	s.keyboardPausePending = false
	s.Paused = false
	s.Sound.Pause(false)
	s.Rules.BeginQuit()
	s.quit = &quitFlow{}
}

func (s *Session) prepareQuitAudio() {
	if s.quit.phase == quitExitStart {
		s.Sound.BeginQuitFade(s.Table.Quit.FadeTarget, s.Table.Quit.FadeStep)
		s.quit.phase = quitAudioFade
	}
}

// advanceQuit owns one original wait. The confirming question keeps its final
// prompt until the exit service begins; cancellation returns to table attract.
func (s *Session) advanceQuit(input Input) quitOutcome {
	confirm, cancel := input.QuitConfirm, input.QuitCancel || input.AnyKey
	q := s.quit
	s.prepareQuitAudio()
	s.beginTableWait(input)
	switch q.phase {
	case quitQuestion:
		s.Rules.ScoreCycleTick()
		s.writePanel(s.Table.Quit.Prompt)
		s.Camera, s.cameraUp = bounceTableCamera(s.Camera, s.cameraUp)
		// The original camera service clears the repeated-note phase left
		// by the preceding lamp service, as it does during score cycling.
		s.Sound.SetRetriggerPhase([16]byte{})
		s.Engine.CaptureVisualNudge()
		if confirm {
			s.keyboardPausePending = false
			q.phase = quitExitStart
		} else if cancel {
			s.keyboardPausePending = false
			q.phase = quitAborted
			return quitCancelled
		}
	case quitAudioFade:
		s.Rules.PreparationTick()
		s.Engine.CaptureVisualNudge()
		if s.Sound.FadeComplete() {
			q.phase = quitPaletteFade
		}
	case quitPaletteFade:
		s.Rules.PreparationTick()
		s.Engine.CaptureVisualNudge()
		q.paletteSteps++
		if q.paletteSteps == s.Table.Quit.PaletteFrames {
			q.phase = quitComplete
			return quitExited
		}
	case quitComplete:
		return quitExited
	case quitAborted:
		return quitCancelled
	}
	return quitPending
}

// quitPalette applies the source RGB444 channel step to base colors. The EHB
// half is derived afterward, retaining its relationship to the faded base.
func (s *Session) quitPalette(original color.Palette, halfBright bool) color.Palette {
	if s.quit == nil || s.quit.paletteSteps == 0 {
		return original
	}
	palette := make(color.Palette, len(original))
	base := len(original)
	if halfBright {
		base /= 2
	}
	for index := 0; index < base; index++ {
		c := color.RGBAModel.Convert(original[index]).(color.RGBA)
		fade := func(channel byte) byte { return byte(max(0, int(channel)/17-s.quit.paletteSteps) * 17) }
		c.R, c.G, c.B = fade(c.R), fade(c.G), fade(c.B)
		palette[index] = c
		if halfBright {
			c.R, c.G, c.B = (c.R/17>>1)*17, (c.G/17>>1)*17, (c.B/17>>1)*17
			palette[base+index] = c
		}
	}
	return palette
}
