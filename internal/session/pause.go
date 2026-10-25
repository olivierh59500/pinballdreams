package session

// tablePause owns the original live-table pause and its quit question. System
// suspension remains a separate freeze, with no waits or spring updates.
type tablePause struct {
	question, exiting bool
	suspended         bool
}

// DefersKeyboardPause reports nested table presentation or ball preparation. The
// original key decoder records P without interrupting the running routine;
// its next phase replaces that request. Host and touch pauses stay immediate.
func (s *Session) DefersKeyboardPause() bool {
	if s.Paused || s.Table == nil || s.NameEntry != nil {
		return false
	}
	return s.entry != nil && !s.entry.ready || s.quit != nil || s.attract != nil || s.gameOver || s.Playing && (s.restoration != nil || s.preparation != nil || s.launching || s.inBonusPresentation())
}

func (s *Session) inBonusPresentation() bool {
	return s.pendingDrain || s.Bonus != nil || s.finale != nil
}

func (s *Session) applyPauseKey(input Input) Input {
	deferred := s.DefersKeyboardPause()
	if input.PauseKeyPressed {
		if s.quit != nil && s.quit.phase == quitQuestion && !s.Paused {
			// Clearing the original P packet makes the question's release
			// matcher take its ordinary cancellation branch in this wait.
			input.QuitCancel = true
		} else if deferred {
			s.keyboardPausePending = true
		} else if s.NameEntry == nil || s.Paused {
			input.Pause = true
		}
	}
	launchRoutine := s.Playing && (s.restoration != nil || s.preparation != nil || s.launching)
	if deferred && (s.inBonusPresentation() || launchRoutine && s.keyboardPausePending) {
		// Bonus presentation does not poll Escape. In the spring routine,
		// a pending P also disables the original cancellation decoder.
		input.Back = false
	}
	return input
}

func (s *Session) pausePressed(input Input) bool {
	return input.Pause && !s.previous.Pause || s.Paused && input.AnyKey
}

func (s *Session) advanceTablePause(input Input, entering bool) {
	s.Sound.StopVoices()
	if s.frameAudio != nil {
		s.frameAudio.waitOnly = true
	}
	s.panel.scoreDigits = 0
	s.beginTableWait(input)
	s.Rules.Frame++
	p := s.tablePause
	if !entering {
		key := input.AnyKey || input.Pause && !s.previous.Pause
		back := input.ExitTable || input.Back && !s.previous.Back
		if p.question {
			if input.QuitConfirm {
				p.question, p.exiting, s.Paused = false, true, false
				s.Sound.Pause(false)
			} else if input.QuitCancel || key || back {
				p.question = false
			}
		} else if back {
			p.question = true
		} else if key {
			s.Paused = false
			s.Sound.Pause(false)
			s.tablePause = nil
		}
	}
	text := s.Table.PauseText
	if p.question {
		text = s.Table.Quit.Prompt
	}
	s.writePanel(text)
}
