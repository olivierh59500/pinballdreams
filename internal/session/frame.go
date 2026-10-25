package session

// beginObservedFrame provides a development observation boundary after phase
// entry actions and before ordinary input/motion. It does not drive live audio.
func (s *Session) beginObservedFrame() {
	if s.observeFrame != nil {
		s.observeFrame()
	}
}
