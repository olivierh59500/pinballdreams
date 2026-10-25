package session

// beginTableWait keeps spring input active while bonus counting, score entry
// or other presentation owns a table wait. These phases discard the release
// impulse while still drawing and releasing the same spring as live play.
func (s *Session) beginTableWait(input Input) {
	s.beginObservedFrame()
	s.sampleTableInput(input)
}

// sampleSpring follows the wait-side input service independently of motion.
// Releases retain the last drawn gauge until the following idle update.
func (s *Session) sampleSpring(input Input) int {
	if s.observeSpring != nil {
		defer s.observeSpring()
	}
	s.springMotion += input.SpringMotion
	if s.previous.Launch && !input.Launch {
		// Key release precedes motion sampling. Unread movement joins the
		// following update, while the previous gauge remains visible.
		return s.releaseSpring()
	}
	// Negation precedes signed sampling, including the shared half-turn
	// boundary. This is a relative input rule, not a mouse-counter model.
	motion := -int(int8(-s.springMotion)) * s.Table.SpringMouseScale
	s.springMotion = 0
	pull := s.Charge + motion
	if input.Launch {
		pull += s.Table.SpringStep
	}
	if pull <= 0 {
		// Crossing rest starts a new pull from the current movement alone.
		pull = -motion
	}
	if pull > s.Table.SpringLimit {
		// Saturation retains the sampled motion's sub-level remainder.
		pull = s.Table.SpringLimit - int(uint8(-motion))
	}
	s.Charge = pull
	s.springLevel = s.Table.SpringGauge.Level(pull)
	s.Rules.DrawSpring(s.springLevel)
	if input.SpringClick {
		return s.releaseSpring()
	}
	return 0
}

func (s *Session) releaseSpring() int {
	s.Sound.ReleaseSpring(s.springLevel)
	impulse := s.Table.SpringImpulse(s.Charge)
	s.Charge = 0
	return impulse
}

// updateSpringFrame follows the source's restricted launch loop. Passage
// geometry and presentation run while ordinary game awards, collectors,
// gravity sampling, cooldowns and tilt accumulation remain idle.
func (s *Session) updateSpringFrame(input Input, impulse int) {
	s.updateCamera()
	s.Engine.Step(s.motionControls(input))
	s.prepareBallSprite()
	if s.ballStatus != nil {
		s.ballStatus.advance(s.Rules.Frame+1, s.PlayerCount, s.Player, s.Ball, s.Table.ScoreText(s.Player, s.Players[s.bankPlayer].Score))
		s.writePanel(s.ballStatus.text)
		if s.ballStatus.score {
			s.panel.number(s.Players[s.bankPlayer].Score)
		}
	}
	s.pollPlayerSelection(input)
	rollovers := false
	if s.Rules.HasRollovers() {
		rollovers = s.Table.RolloverContact(s.Rules.ArtView(false, false), s.Engine.Layer,
			s.Engine.State.PixelX(), s.Engine.State.PixelY(), s.Engine.State.Nudge())
	}
	s.Rules.SpringSensorContacts(s.Engine.State.PixelX()+8, s.Engine.State.PixelY()+8, rollovers)
	s.Engine.SelectLayer()
	s.Rules.Layer = s.Engine.Layer
	s.Rules.PreparationTick()
	s.Engine.Locked = s.Rules.Locked
	s.InChute = s.Rules.SensorActive(s.Table.Launch.SpringSensor)
	if s.Rules.SensorActive(s.Table.Launch.PlaySensor) {
		s.keyboardPausePending = false
		s.panel.idle = s.Table.Rules.Panel.InitialDelay
		s.launching = false
		s.playerSelectionOpen = false
		s.Sound.BeginPlay(s.MusicDisabled)
		s.Rules.ConsumeRelaunchCredit()
		s.awaitingPlayMusic = false
	} else if s.InChute && input.Back && !s.previous.Back {
		s.keyboardPausePending = false
		// The source cancels only inside its spring detector, after the
		// current motion/sensor wait. Reaching play takes precedence.
		s.Playing, s.launching = false, false
		s.spritesHidden = true
		s.PlayerCount = 0
		s.pendingTableReset = true
		s.Sound.Trigger(s.Table.Launch.CancelCue)
	} else if s.InChute && impulse > 0 {
		s.Engine.State.VY -= int16(impulse)
	}
}

func (s *Session) updateCamera() {
	s.Engine.CaptureVisualNudge()
	// Both source play loops scroll from the preceding ball pose before motion.
	y := s.Engine.State.PixelY()
	delta := (y - s.Camera - 90) / 7
	if delta < 0 && s.Camera <= 64 {
		delta = max(delta, -s.Camera)
	}
	if s.Camera >= 212 {
		delta = min(delta, 276-s.Camera)
	}
	if s.cameraTransition {
		if delta >= -3 && delta <= 3 {
			s.finishCameraTransition()
			return
		}
		delta = max(-3, min(3, delta))
	}
	s.Camera += delta
	if s.Camera <= 0 {
		s.Camera = 0
		s.finishCameraTransition()
	} else {
		s.Camera = min(276, s.Camera)
	}
}

func (s *Session) finishCameraTransition() {
	if !s.Rules.CaptureContact() {
		s.cameraTransition = false
		s.Engine.Locked = false
	}
}
