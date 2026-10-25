package session

// ApplyPointer maps native canvas coordinates to semantic game controls.
// Multiple current touches contribute to the same input value.
func (s *Session) ApplyPointer(i *Input, x, y, height int) {
	width := 320
	if s.Intro != nil {
		width = s.Intro.Size().X
	}
	if x < 0 || y < 0 || x >= width || y >= height {
		return
	}
	if s.Table == nil {
		if s.Paused {
			i.Pause = true
		} else if s.Intro != nil {
			i.Skip = true
		} else if s.Scores != nil {
			i.HighScores = true
		} else {
			i.Table = s.Menu.TableAt(x, y-(height-256)/2)
			if i.Table == 0 && y >= (height-256)/2+236 && y < (height+256)/2 {
				i.HighScores = true
			}
		}
		return
	}
	if s.AskingToQuit() && (!s.Paused || s.tablePause != nil) {
		ui := &s.Table.Quit
		if y < 23 && x >= ui.ConfirmCell*16 && x < (ui.ConfirmCell+1)*16 {
			i.QuitConfirm = true
		} else {
			i.QuitCancel = true
		}
		return
	}
	if s.Paused {
		i.Pause = true
		return
	}
	if s.NameEntry != nil {
		if x < 120 {
			i.Left = true
		} else if x > 200 {
			i.Right = true
		} else {
			i.Launch = true
		}
		return
	}
	if y < s.Table.Viewport.PanelHeight && x < 8*16 && s.playerSelectionOpen && s.PlayerCount > 0 && !s.gameOver {
		// The existing player prefix doubles as the native add-player control.
		// A zero count requests the source Return-key increment, capped at eight.
		i.Start = true
		i.Players = 0
		return
	}
	if !s.Playing && !s.PreparingBall() {
		i.Start = true
		i.Players = 1
		return
	}
	if y < s.Table.Viewport.PanelHeight {
		if x >= s.Table.Viewport.Width-32 {
			i.Music = true
		} else {
			i.Pause = true
		}
		return
	}
	// The launch lane is directly touchable where the original spring is drawn.
	// Give it priority over the right flipper's broad touch region.
	if x >= 260 && y >= height-100 {
		i.Launch = true
		return
	}
	if x < 120 {
		i.Left = true
	} else if x > 200 {
		i.Right = true
	} else if y > height/2 {
		i.Launch = true
	} else {
		i.Nudge = true
	}
}
