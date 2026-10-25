package session

// enterTableAttract follows the original record reset after game over or a
// cancelled exit. Physical motion and the shared pan/animation clocks survive.
func (s *Session) enterTableAttract() {
	s.keyboardPausePending = false
	s.Players = [8]Player{}
	// The selected mask returns to player zero, while the renderer's actor
	// cache remains owned by the final bank until the next spring entry.
	s.Player, s.PlayerCount, s.Ball = 0, 0, 1
	s.Rules.ResetTableAttract()
	s.Rules.UpdateRankingTemplates(s.ScoreBook[s.Table.ID])
	s.Rules.PlayerCount = 0
	s.Rules.Locked, s.Engine.Locked = false, false
	s.Playing, s.Paused, s.InChute = false, false, false
	s.spritesHidden = true
	s.playerSelectionOpen = true
	s.startPhase = tableStartIdle
	s.pendingTableReset, s.pendingGameOver, s.gameOver = false, false, false
	s.pendingNewBall, s.pendingDrain, s.pendingReplacement = false, false, false
	s.preparation, s.restoration, s.ballStatus = nil, nil, nil
	s.Bonus, s.finale, s.NameEntry = nil, nil, nil
	s.launching, s.tiltEntered, s.cameraTransition = false, false, false
	s.bonusClearPlayer = -1
	if s.attract == nil {
		s.attract = newTableAttract(&s.Table.Attract, s.Rules, s.ScoreBook[s.Table.ID])
	} else {
		s.attract.Reset(s.Rules, s.ScoreBook[s.Table.ID])
	}
}

func (s *Session) updateTableAttract(input Input) {
	s.beginTableWait(input)
	frame := s.attract.Tick(s.Camera, s.cameraUp)
	s.Camera, s.cameraUp = frame.Camera, frame.Upward
	s.Engine.CaptureVisualNudge()
	if frame.Wrote {
		s.writePanel(frame.Text)
	}
}
