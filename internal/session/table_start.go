package session

type tableStartPhase int

const (
	tableStartIdle tableStartPhase = iota
	tableStartEntering
	tableStartPanning
)

// selectTablePlayers handles a source function-key selection after the current
// table wait. Restarting an ended game first resets its actor/rule records.
func (s *Session) selectTablePlayers(players int, restart bool) {
	first := s.PlayerCount == 0 || restart
	previous := s.PlayerCount
	if first {
		previous = 0
	}
	if players == 0 {
		if previous == 8 {
			return
		}
		players = previous + 1
	}
	if restart {
		s.enterTableAttract()
	}
	s.PlayerCount = max(1, min(8, players))
	s.Rules.PlayerCount = s.PlayerCount
	s.playerSelectionOpen = true
	if first {
		s.Sound.Trigger(s.Table.Attract.FirstStartCue)
		s.Sound.StopAtReturn()
	} else {
		s.Sound.Trigger(s.Table.Attract.StartCue)
	}
	if restart {
		// State six still performs its initial roaming wait before panning.
		s.startPhase = tableStartEntering
	} else if s.startPhase == tableStartIdle && s.attract != nil {
		s.beginTablePan()
	}
}

func (s *Session) pollPlayerSelection(input Input) {
	if s.playerSelectionOpen && s.startPressed(input) {
		s.selectTablePlayers(input.Players, false)
	}
}

func (s *Session) startPressed(input Input) bool {
	return input.StartPressed || input.Start && !s.previous.Start
}

func (s *Session) beginTablePan() {
	s.startPhase = tableStartPanning
	s.cameraUp = false
	s.writePanel(s.Table.BallPreparation.BlankPrompt)
	s.ballStatus = &ballStatus{assets: &s.Table.BallPreparation, remaining: 1}
}

// updateTableStart pans toward the spring before dispatching ordinary serving.
// The physical ball stays hidden and stationary throughout this source phase.
func (s *Session) updateTableStart(input Input) {
	if s.startPhase == tableStartEntering {
		s.updateTableAttract(input)
		s.beginTablePan()
		return
	}
	s.beginTableWait(input)
	s.Camera, s.cameraUp = startTableCamera(s.Camera, s.cameraUp)
	s.Engine.CaptureVisualNudge()
	s.Rules.ScoreCycleTick()
	s.writePanel(s.ballStatus.advance(s.Rules.Frame, s.PlayerCount, 0, 1, s.Table.ScoreText(0, s.Players[0].Score)))
	if s.Camera == 276 {
		s.keyboardPausePending = false
		s.Rules.PrepareGame()
		s.attract = nil
		s.startPhase = tableStartIdle
		s.Playing = true
		s.pendingNewBall = true
	}
}
