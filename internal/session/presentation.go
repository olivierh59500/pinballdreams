package session

import (
	"github.com/olivierh59500/pinballdreams/internal/frontend"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

func (s *Session) playMusic(module *sound.Module) {
	lowPass := s.presentation.InitialLowPass
	if s.Sound != nil {
		lowPass = s.Sound.LowPassEnabled()
	}
	s.Sound = sound.New(module, module.InitialOrder)
	s.Sound.SetLowPass(lowPass)
	s.Sound.Mute(s.Muted)
	s.AudioChanged = true
}

// ShowMenu selects the dedicated menu song for direct entry or a table return.
// Finishing the opening instead keeps its running song through enterMenu.
func (s *Session) ShowMenu() error {
	music, err := sound.DecodeFrontend(s.Source, false)
	if err != nil {
		return err
	}
	s.enterMenu()
	s.playMusic(music)
	return nil
}

// enterMenu resets the presentation layout while retaining the current music.
func (s *Session) enterMenu() {
	s.Table = nil
	s.Engine = nil
	s.Rules = nil
	s.Bonus = nil
	s.NameEntry = nil
	s.quit = nil
	s.tablePause = nil
	s.keyboardPausePending = false
	s.attract = nil
	s.startPhase = tableStartIdle
	s.playerSelectionOpen = false
	s.gameOver, s.pendingTableReset, s.pendingGameOver = false, false, false
	s.Scores = nil
	s.scoresPending = false
	s.attractFrames = 0
	s.sceneChangeWait = false
	s.selection = nil
	s.entry = nil
	s.Intro = nil
	s.Menu = frontend.NewMenu(&s.presentation.Menu)
	s.footer = frontend.NewFooter(&s.presentation.Footer)
	s.footerStarted = false
	s.Playing, s.Paused = false, false
	s.Sound.SetRetriggerPhase([16]byte{})
}

func (s *Session) updatePresentation(input Input) error {
	if s.pausePressed(input) {
		s.Paused = !s.Paused
		s.Sound.Pause(s.Paused)
	}
	if s.Paused {
		return nil
	}
	if s.selection != nil {
		return s.updateTableSelection(input)
	}
	s.advanceAudio()
	if s.Intro != nil {
		// Direct table keys are a host shortcut, like the desktop -table flag.
		if input.Table >= 1 && input.Table <= 4 {
			return s.Select(input.Table - 1)
		}
		skip := input.Skip && !s.previous.Skip || s.startPressed(input) || input.Back && !s.previous.Back
		s.Intro.Tick(frontend.OpeningInput{Skip: skip, ExtraCredits: input.IntroExtraCredits})
		if s.Intro.Done() {
			s.enterMenu()
		}
		return nil
	}
	if s.footerStarted || s.Menu.Ready() {
		s.footerStarted = true
		s.footer.Tick()
	}
	// Direct table keys remain available throughout the ranking transition.
	if s.Scores != nil && input.Table >= 1 && input.Table <= 4 {
		s.selectFromMenu(input.Table)
		return nil
	}
	// Both original scene changes keep one dark update between their fades.
	// The shared footer continues advancing through that wait.
	if s.sceneChangeWait {
		s.sceneChangeWait = false
		// Completing the ranking layout selects its supplied repeat cadence.
		// Returning to table selection clears it after the same dark wait.
		phase := [16]byte{}
		if s.Scores != nil {
			phase = s.presentation.Scores.MusicRetriggerPhase
		}
		s.Sound.SetRetriggerPhase(phase)
		return nil
	}
	if s.Scores != nil {
		ready := s.Scores.Ready()
		s.Scores.Tick()
		if ready && s.Scores.Ready() {
			s.attractFrames++
			if s.attractFrames < s.presentation.Scores.HoldFrames {
				key := input.MenuKey
				if key == 0 && (input.HighScores && !s.previous.HighScores || input.Back && !s.previous.Back || s.startPressed(input)) {
					// Touch and platform navigation follow the same matcher
					// as a non-letter key, including its musical side effect.
					key = OtherMenuKey
				}
				s.footer.KeyPress(key)
				if key != 0 {
					s.Sound.SetRetriggerPhase([16]byte{})
				}
			}
			if s.attractFrames >= s.presentation.Scores.HoldFrames || input.HighScores && !s.previous.HighScores || input.Back && !s.previous.Back || s.startPressed(input) {
				s.Scores.Hide()
			}
		}
		if s.Scores.Hidden() {
			s.Scores = nil
			s.Menu = frontend.NewMenu(&s.presentation.Menu)
			s.attractFrames = 0
			s.sceneChangeWait = true
		}
		return nil
	}
	ready := s.Menu.Ready()
	s.Menu.Tick()
	if s.Menu.Ready() && input.Table >= 1 && input.Table <= 4 {
		s.selectFromMenu(input.Table)
		return nil
	}
	if ready && s.Menu.Ready() && !s.scoresPending {
		s.attractFrames++
		if s.attractFrames >= s.presentation.Menu.HoldFrames || input.HighScores && !s.previous.HighScores {
			s.Menu.Hide()
			s.scoresPending = true
		}
	}
	if s.scoresPending && s.Menu.Hidden() {
		s.Scores = frontend.NewScores(&s.presentation.Scores, s.ScoreBook)
		s.scoresPending = false
		s.attractFrames = 0
		s.sceneChangeWait = true
	}
	return nil
}
