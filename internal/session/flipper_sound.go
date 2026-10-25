package session

// sampleFlipperSounds handles fresh keys while ordinary table controls are
// enabled. Serving keeps these sounds active even though physics is frozen.
// Restoration, bonus counting and table exit do not accept flipper clicks.
func (s *Session) sampleFlipperSounds(input Input) {
	if !s.tableControlsEnabled() {
		return
	}
	if input.Left && !s.previous.Left {
		s.Sound.Flipper()
	}
	if input.Right && !s.previous.Right {
		s.Sound.Flipper()
	}
}
