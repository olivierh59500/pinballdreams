package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestNavigationInspectionDistinguishesQuestionAndStoppedFade(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Select(table); err != nil {
				t.Fatal(err)
			}
			s.Sound.BeginTilt(s.Table.Tilt.EntryCue)
			for waits := 0; !s.Sound.PlaybackStopped(); waits++ {
				if waits >= 20000 {
					t.Fatal("original terminal song did not stop")
				}
				s.Sound.AdvanceFrame()
			}
			s.quit = &quitFlow{phase: quitQuestion}
			state := s.NavigationState()
			if state.Table != table+1 || state.ExitPhase != "question" || !state.AudioStopped || state.ExitBlocked {
				t.Fatalf("unanswered question misclassified: %+v", state)
			}
			s.quit.phase = quitAudioFade
			s.Sound.BeginQuitFade(0, 1)
			clock, outputs := s.Rules.Frame, s.Sound.OutputDeclarations()
			for range 10 {
				state = s.NavigationState()
				if state.ExitPhase != "audio-fade" || !state.ExitBlocked || state.FadeComplete {
					t.Fatalf("stopped fade misclassified: %+v", state)
				}
			}
			if s.Rules.Frame != clock || s.Sound.OutputDeclarations() != outputs || !s.Sound.PlaybackStopped() {
				t.Fatal("inspection changed the game or its terminal-song policy")
			}
			s.Sound.BeginPlay(false)
			if state := s.NavigationState(); state.AudioStopped || state.ExitBlocked {
				t.Fatalf("running fade misclassified: %+v", state)
			}
		})
	}
}
