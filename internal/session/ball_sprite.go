package session

// ballSprite keeps the last prepared mask independently of the current ball
// pose. Captures can move the ball or select another cover after preparation;
// readiness can enable the retained sprite before its next mask update.
type ballSprite struct {
	pixels [16 * 15]byte
}

// PreparedBallPixels snapshots the renderer's retained sprite mask. A capture
// or layer change can follow preparation without replacing those pixels.
func (s *Session) PreparedBallPixels() [16 * 15]byte { return s.ballSprite.pixels }

func (s *Session) prepareBallSprite() {
	state := &s.Engine.State
	for y := range 15 {
		for x := range 16 {
			s.ballSprite.pixels[y*16+x] = s.Table.VisibleBallIndex(s.Engine.Layer, state.PixelX(), state.PixelY(), state.Nudge(), x, y)
		}
	}
}
