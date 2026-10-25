package physics

const (
	nudgeOutwardSpeed = 600
	nudgeReturnSpeed  = 200
	nudgeTravelLimit  = 2048
	flipperLiftSpeed  = 65
	flipperLiftAccel  = 7
	flipperFallAccel  = 5
	flipperFrameStep  = 55
	flipperLastFrame  = 16
)

// Integrate includes the source flipper acceleration and one-unit spin decay.
func (e *Engine) Integrate() {
	s := &e.State
	if e.Controls.Nudge {
		s.NudgeSpeed = nudgeOutwardSpeed
		s.NudgePosition += nudgeOutwardSpeed
		if s.NudgePosition >= nudgeTravelLimit {
			s.NudgePosition = nudgeTravelLimit
			s.NudgeSpeed = 0
		}
	} else {
		s.NudgeSpeed = -nudgeReturnSpeed
		s.NudgePosition -= nudgeReturnSpeed
		if s.NudgePosition < 0 {
			s.NudgePosition = 0
			s.NudgeSpeed = 0
		}
	}
	s.LeftPhase = flipper(&s.LeftSpeed, &s.LeftPosition, e.Controls.Left && !e.Tilted)
	s.RightPhase = flipper(&s.RightSpeed, &s.RightPosition, e.Controls.Right && !e.Tilted)
	if e.Locked {
		return
	}
	s.X += int32(s.VX)
	s.Y += int32(s.VY)
	s.VX += e.GravityX
	s.VY += e.GravityY
	if s.Spin > 0 {
		s.Spin--
	} else if s.Spin < 0 {
		s.Spin++
	}
}

func flipper(speed, position *int16, pressed bool) int16 {
	if pressed {
		*speed -= flipperLiftAccel
		if *speed > -flipperLiftSpeed {
			*speed = -flipperLiftSpeed
		}
	} else {
		*speed += flipperFallAccel
	}
	*position -= *speed
	phase := int16(int32(*position) / flipperFrameStep)
	if phase <= 0 {
		*speed = 0
		*position = flipperFrameStep
		phase = 0
	}
	if phase >= flipperLastFrame {
		*speed = 0
		*position = flipperLastFrame * flipperFrameStep
		phase = flipperLastFrame
	}
	return phase
}
