package session

// updateScoreCycle services a source game-over wait. Text and lamps advance;
// the camera bounces over the table while physical motion remains stationary.
func (s *Session) updateScoreCycle(input Input) {
	s.presentScoreCycle(input)
	s.finishScoreCycle()
}

// presentScoreCycle ends at the keyboard poll. Restart and quit requests
// branch here, before advancing the player or the repetition counters.
func (s *Session) presentScoreCycle(input Input) {
	s.beginTableWait(input)
	ui := &s.Table.ScoreEntry
	cycle := ui.GameOverDelay + ui.PlayerDelay
	player := s.gameOverFrames / cycle % s.PlayerCount
	s.Player, s.bankPlayer, s.Rules.Player = player, player, player
	s.Rules.ScoreCycleTick()
	s.Camera, s.cameraUp = bounceTableCamera(s.Camera, s.cameraUp)
	// The score-cycle camera service clears the inherited bank cadence
	// before the next replay tick, even when no lamp bank is collected.
	s.Sound.SetRetriggerPhase([16]byte{})
	s.Engine.CaptureVisualNudge()
	if s.gameOverFrames%cycle < ui.GameOverDelay {
		s.writePanel(ui.GameOverText)
	} else {
		s.writeScorePanel(player, s.Players[player].Score)
	}
}

func (s *Session) finishScoreCycle() {
	ui := &s.Table.ScoreEntry
	cycle := ui.GameOverDelay + ui.PlayerDelay
	player := s.gameOverFrames / cycle % s.PlayerCount
	s.gameOverFrames++
	finished := s.gameOverFrames == cycle*s.PlayerCount*ui.Repeats
	if s.gameOverFrames%cycle == 0 {
		next := player + 1
		if next == s.PlayerCount && !finished {
			next = 0
		}
		s.Player = next
		if next < s.PlayerCount {
			s.bankPlayer, s.Rules.Player = next, next
		}
	}
	if finished {
		s.keyboardPausePending = false
		s.gameOver = false
		s.pendingTableReset = true
	}
}
