package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/physics"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestFirstFrameImpactHasNoArtificialScoreDelay(t *testing.T) {
	// The same source-observed left-slingshot motion produces these rewards
	// on the later tables. A session-wide four-frame cooldown lost the hit.
	for _, c := range []struct {
		table int
		score uint64
	}{{1, 15}, {2, 1000}, {3, 15}} {
		t.Run(resources.Names[c.table], func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Select(c.table); err != nil {
				t.Fatal(err)
			}
			s.Start(1)
			waitForPreparedBall(t, s)
			// This source impact fixture begins after the launch-loop exit.
			s.launching = false
			s.Engine.State.X, s.Engine.State.Y = 70<<10, 400<<10
			s.Engine.State.VX, s.Engine.State.VY = -1800, 1200
			var cues []audiodata.Command
			s.Rules.OnSound = func(cue audiodata.Command) { cues = append(cues, cue) }
			if err := s.Update(Input{Left: true, Right: true}); err != nil {
				t.Fatal(err)
			}
			if s.Engine.ImpactObject == 0 || s.Players[0].Score != c.score || s.Players[0].Bonus != 0 || len(cues) != 1 {
				t.Fatalf("first-frame reward differs from the source: impact=%x player=%+v cues=%+v", s.Engine.ImpactObject, s.Players[0], cues)
			}
		})
	}
}

func TestCanceledSubstepImpactDoesNotAwardPointsOrPlayItsCue(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	waitForPreparedBall(t, s)
	s.launching = false
	for range 5 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	// The original frame briefly arms Ignition's left slingshot, then a
	// later response cancels it. Publishing from a substep awards it too soon.
	s.Engine = physics.New(s.Table)
	s.Engine.State.X, s.Engine.State.Y = 70<<10, 400<<10
	s.Engine.State.VX, s.Engine.State.VY = -1800, 1200
	var cues []audiodata.Command
	s.Rules.OnSound = func(cue audiodata.Command) { cues = append(cues, cue) }
	if err := s.Update(Input{Left: true, Right: true}); err != nil {
		t.Fatal(err)
	}
	if s.Engine.State.X != 81287 || s.Engine.State.Y != 407448 || s.Engine.State.VX != 2342 || s.Engine.State.VY != -490 {
		t.Fatalf("canceled-impact trajectory differs from the original: %+v", s.Engine.State)
	}
	if s.Engine.ImpactObject != 0 || s.Engine.FrameContact.Powered || s.Players[0].Score != 0 || s.Players[0].Bonus != 0 || len(cues) != 0 {
		t.Fatalf("canceled impact reached gameplay: contact=%+v player=%+v cues=%+v", s.Engine.FrameContact, s.Players[0], cues)
	}
}

func TestMovingFlipperContactSurvivesLaterStationarySubstep(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(2); err != nil {
		t.Fatal(err)
	}
	s.EnableFrameAudio(nil)
	if err := s.Update(Input{Start: true, Players: 1, Launch: true}); err != nil {
		t.Fatal(err)
	}
	for range 400 {
		if err := s.Update(Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	for frame := 1; frame <= 187; frame++ {
		if err := s.Update(Input{Right: frame >= 185}); err != nil {
			t.Fatal(err)
		}
	}
	contact := s.Engine.FrameContact
	if !s.Engine.MovingFlipperContact || !s.Engine.HadContact || contact.Material != 2 || contact.SurfaceX != 0 || contact.SurfaceY != 0 {
		t.Fatalf("early moving contact lost behind final contact: %+v", contact)
	}
	if err := s.Update(Input{Right: true}); err != nil {
		t.Fatal(err)
	}
	if s.Engine.MovingFlipperContact {
		t.Fatal("moving contact leaked into the next update")
	}
}
