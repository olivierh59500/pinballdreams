package session

// advanceNudge runs after live-frame awards and collectors. The tilt edge
// returns before copying the previous input or advancing its decay counter.
func (s *Session) advanceNudge(held bool) {
	ui := &s.Table.Tilt
	if held != s.nudgePrevious {
		s.nudgeMeter = s.nudgeMeter<<1 | 1
		if ui.WarningMeter != 0 && int(s.nudgeMeter) == ui.WarningMeter {
			s.Sound.Trigger(ui.Warning)
		}
		if int(s.nudgeMeter) == ui.TiltMeter {
			s.Engine.Tilted = true
			return
		}
	}
	s.nudgePrevious = held
	s.nudgeDecay = int(uint16(s.nudgeDecay - 1))
	if s.nudgeDecay == 0 {
		s.nudgeDecay = ui.DecayFrames
		s.nudgeMeter >>= 1
	}
}

func (s *Session) enterTilt() {
	s.tiltEntered = true
	s.awaitingPlayMusic = false
	s.Engine.Locked = false
	s.Rules.BeginTilt()
	s.Sound.BeginTilt(s.Table.Tilt.EntryCue)
}

// updateTiltFrame retains the source's separate presentation/motion loop.
// Live table actions, sensor awards and collectors do not run after tilt.
func (s *Session) updateTiltFrame(input Input) {
	if !s.tiltEntered {
		s.enterTilt()
	}
	s.Rules.BeginLiveFrame()
	s.updateCamera()
	s.Engine.Step(s.motionControls(input))
	s.prepareBallSprite()
	s.writePanel(s.Table.Tilt.Text)
	s.Engine.SelectLayer()
	s.Rules.Layer = s.Engine.Layer
	s.Rules.FinishTiltFrame()
	s.Engine.Locked = false
	s.InChute = false
	if s.Engine.State.PixelY() > 512 {
		s.pendingDrain = true
	}
}
