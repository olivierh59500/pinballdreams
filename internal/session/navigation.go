package session

// NavigationState is a read-only diagnostic view, sampled on the game thread.
// It distinguishes an unanswered question from the original terminal-song
// fade wait without assigning any state or changing the exit policy.
type NavigationState struct {
	Screen, ExitPhase                           string
	Table, Clock                                int
	Playing, Paused, AudioStopped, FadeComplete bool
	ExitBlocked                                 bool
}

func (s *Session) NavigationState() NavigationState {
	state := NavigationState{Screen: "menu", Playing: s.Playing, Paused: s.Paused}
	if s.Intro != nil {
		state.Screen = "opening"
	}
	if s.Table != nil {
		state.Screen, state.Table = "table", s.Table.ID+1
	}
	if s.Rules != nil {
		state.Clock = s.Rules.Frame
	}
	if s.quit != nil {
		state.ExitPhase = [...]string{"question", "exit-start", "audio-fade", "palette-fade", "complete", "aborted"}[s.quit.phase]
	} else if s.tablePause != nil && s.tablePause.question {
		state.ExitPhase = "paused-question"
	}
	if state.ExitPhase != "" && s.Sound != nil {
		state.AudioStopped = s.Sound.PlaybackStopped()
		state.FadeComplete = s.Sound.FadeComplete()
		state.ExitBlocked = state.ExitPhase == "audio-fade" && state.AudioStopped && !state.FadeComplete
	}
	return state
}
