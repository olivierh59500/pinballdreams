package game

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

func TestRotationSilentlyCancelsSpringDuringEntryAndAttract(t *testing.T) {
	for table, name := range resources.Names {
		for _, phase := range []string{"entry", "attract", "start-pan"} {
			t.Run(name+"/"+phase, func(t *testing.T) {
				g, err := New(nil, true)
				if err != nil {
					t.Fatal(err)
				}
				g.height = resources.FullTableHeight
				if err = g.Session.ShowMenu(); err != nil {
					t.Fatal(err)
				}
				for !g.Session.Menu.Ready() {
					if err = g.Session.Update(session.Input{}); err != nil {
						t.Fatal(err)
					}
				}
				if err = g.Session.Update(session.Input{Table: table + 1}); err != nil {
					t.Fatal(err)
				}
				for waits := 0; g.Session.Table == nil; waits++ {
					if waits >= 512 {
						t.Fatal("public menu selection did not reach table entry")
					}
					if err = g.Session.Update(session.Input{Launch: true}); err != nil {
						t.Fatal(err)
					}
				}
				if phase != "entry" {
					// Let the ordinary palette entry finish and begin roaming
					// before a public held-launch request starts its camera pan.
					for range 32 {
						if err = g.Session.Update(session.Input{}); err != nil {
							t.Fatal(err)
						}
					}
				}
				for range 8 {
					input := session.Input{Launch: true}
					if phase == "attract" {
						// Free mouse travel pulls during ordinary roaming without
						// invoking the held-key shortcut that starts a game.
						input = session.Input{SpringMotion: 20}
					}
					if err = g.Session.Update(input); err != nil {
						t.Fatal(err)
					}
				}
				if g.Session.Playing || g.Session.Charge == 0 {
					t.Fatal("fixture did not retain a public non-playing spring pull")
				}
				g.pending.add(session.Input{Launch: true, SpringMotion: 100, SpringClick: true}, nil)
				g.controls.canvasHeight = resources.FullTableHeight
				var cues []sound.Command
				g.Session.Sound.ObserveCommands(func(command sound.Command) { cues = append(cues, command) })
				clock, camera, ball := g.Session.Rules.Frame, g.Session.Camera, g.Session.Engine.State
				players, count := g.Session.Players, g.Session.PlayerCount
				g.OrientationChanged()
				if !g.Session.Paused || g.Session.Charge != 0 || !g.controls.waitForRelease || g.pending.input != (session.Input{}) {
					t.Fatal("rotation retained the non-playing spring or pending release")
				}
				// These are host snapshot inputs after remapping and removal,
				// not assigned native physics or source outcome state.
				for _, contacts := range [][]pointerContact{{touch(7, 180, 180, false)}, nil} {
					input := g.controls.apply(g.Session, session.Input{}, contacts, false, 262)
					if err = g.Session.Update(input); err != nil {
						t.Fatal(err)
					}
				}
				if len(cues) != 0 || g.Session.Charge != 0 || g.Session.Engine.State != ball || g.Session.Rules.Frame != clock || g.Session.Camera != camera || g.Session.Players != players || g.Session.PlayerCount != count {
					t.Fatal("rotation/removal emitted a cue or changed frozen table state")
				}
				g.Session.ResumeInteraction()
				if err = g.Session.Update(session.Input{}); err != nil {
					t.Fatal(err)
				}
				if len(cues) != 0 || g.Session.Charge != 0 || g.Session.Engine.State != ball {
					t.Fatal("resuming converted rotation cancellation into a spring impulse")
				}
			})
		}
	}
}

func TestRootOrientationDoesNotPauseMenu(t *testing.T) {
	g, err := New(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err = g.Session.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	g.OrientationChanged()
	if g.Session.Paused || g.controls.waitForRelease {
		t.Fatal("rotation suspended a menu with no table interaction")
	}
}
