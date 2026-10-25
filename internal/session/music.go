package session

// applyMusicToggle handles the original release-side keyboard command and
// fresh pointer events after phase-entry actions, before spring and replay.
func (s *Session) applyMusicToggle(input Input) {
	available := s.tableControlsEnabled() && (!s.Paused || s.tablePause != nil && !s.tablePause.suspended)
	requested := input.MusicPressed || input.Music && !s.previous.Music && !s.Paused
	if available && requested {
		s.MusicDisabled = !s.MusicDisabled
		s.Sound.BeginPlay(s.MusicDisabled)
	}
}
