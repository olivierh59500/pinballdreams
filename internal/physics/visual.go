package physics

// visualPose retains the source art-copy phase independently of the final
// collision pose. Moving frames draw after the first substep; frozen frames
// draw after all four flipper/nudge integrations.
type visualPose struct {
	left, right int16
	nudge       int
	valid       bool
	nudgeValid  bool
}

func (e *Engine) recordVisualPose() {
	e.visual.left, e.visual.right, e.visual.valid = e.State.LeftPhase, e.State.RightPhase, true
}

// CaptureVisualNudge records the row displacement published by the camera.
// Camera-only serving/scrolling updates need this even without ball motion.
func (e *Engine) CaptureVisualNudge() {
	e.visual.nudge, e.visual.nudgeValid = e.State.Nudge(), true
}

// VisualNudge is the table displacement published before this frame's motion.
// Sprite cover sampling continues to use the final physical State.Nudge.
func (e *Engine) VisualNudge() int {
	if e.visual.nudgeValid {
		return e.visual.nudge
	}
	return e.State.Nudge()
}

// VisualFlipperFrame returns the pose last copied by the source art service.
// Before a simulation step, explicit capture poses use their supplied state.
func (e *Engine) VisualFlipperFrame(right bool) int {
	if e.visual.valid {
		if right {
			return int(e.visual.right)
		}
		return int(e.visual.left)
	}
	if right {
		return e.State.RightFrame()
	}
	return e.State.LeftFrame()
}
