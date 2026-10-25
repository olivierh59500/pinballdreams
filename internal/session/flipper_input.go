package session

import "github.com/olivierh59500/pinballdreams/internal/physics"

// flipperInput separates physical holds from accepted game controls. A press
// made while controls are disabled stays blocked until that side is released.
type flipperInput struct {
	left, right               bool
	blockedLeft, blockedRight bool
}

func (f *flipperInput) sample(left, right, enabled bool) {
	f.blockedLeft = left && (f.blockedLeft || !enabled)
	f.blockedRight = right && (f.blockedRight || !enabled)
	f.left = enabled && left && !f.blockedLeft
	f.right = enabled && right && !f.blockedRight
}

func (s *Session) motionControls(input Input) physics.Controls {
	return physics.Controls{Left: s.flippers.left, Right: s.flippers.right, Nudge: input.Nudge}
}
