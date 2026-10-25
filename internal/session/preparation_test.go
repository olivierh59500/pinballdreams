package session

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestPreparationHoldsMotionWhileAllowingSourceSpringCharge(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(1)
			before := s.Engine.State
			// The source restores five Ignition and eight Steel Wheel targets
			// before its two countdown intervals; the other banks are stationary.
			frames := []int{5, 8, 0, 0}[id] + s.Table.BallPreparation.HoldFrames[0] + s.Table.BallPreparation.HoldFrames[1]
			for frame := 0; frame < frames; frame++ {
				if err := s.Update(Input{Launch: true, Left: true, Right: true}); err != nil {
					t.Fatal(err)
				}
				if s.Engine.State != before || s.Charge != min(s.Table.SpringLimit, (frame+1)*s.Table.SpringStep) {
					t.Fatal("preparation changed motion or did not sample the spring input")
				}
				if frame+1 < frames && !s.PreparingBall() {
					t.Fatal("preparation ended before its supplied second interval")
				}
			}
			if s.PreparingBall() {
				t.Fatal("preparation did not end after both supplied intervals")
			}
			if err := s.Update(Input{Launch: true}); err != nil {
				t.Fatal(err)
			}
			if s.Charge != s.Table.SpringLimit {
				t.Fatal("the live spring did not retain its preparation charge")
			}
		})
	}
}

func TestPreparationReleaseCannotLeakAnImpulseIntoSpringPlay(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	before := s.Engine.State
	for range 20 {
		if err := s.Update(Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if !s.PreparingBall() || s.Charge != 0 || s.Engine.State != before {
		t.Fatal("readiness release propelled the ball or retained the charge")
	}
	waitForPreparedBall(t, s)
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.Engine.State.VY < 0 {
		t.Fatal("a released readiness impulse survived the spring-loop entry")
	}
}

func TestNextBallRestoresDroppedGeometryOneFrameAtATime(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	waitForPreparedBall(t, s)
	var targets []int
	for _, group := range s.Table.Rules.TargetGroups {
		for _, id := range group.Targets {
			target := s.Table.Rules.Targets[id]
			if target.Kind == 1 || target.Kind == 8 {
				s.Rules.Targets((target.X0+target.X1)/2, (target.Y0+target.Y1)/2, 6, false)
				targets = append(targets, id)
			}
		}
	}
	if len(targets) != 5 {
		t.Fatal("fixture does not cover all five original drop targets")
	}
	dropped := append([]byte(nil), s.Engine.DynamicCollision...)
	s.nextBall()
	if !bytes.Equal(s.Engine.DynamicCollision, dropped) || s.preparation != nil {
		t.Fatal("new ball restored geometry or started the countdown before the first target frame")
	}
	for n, id := range targets {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
		for _, change := range s.Table.TargetPatches[id] {
			dropped[change.Index] = dropped[change.Index]&^change.Mask | change.Standing
		}
		if !bytes.Equal(s.Engine.DynamicCollision, dropped) {
			t.Fatalf("restoration frame %d did not update exactly its target geometry", n)
		}
		if n+1 < len(targets) && s.preparation != nil {
			t.Fatal("countdown started while targets were still returning")
		}
	}
	if s.restoration != nil || s.preparation == nil || s.preparation.remaining != 50 {
		t.Fatal("last target frame did not start the complete first countdown interval")
	}
}

func TestPauseFreezesPreparationCountdownAndText(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	for range 8 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	remaining, text := s.preparation.remaining, s.ballStatus.text
	for range 25 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.preparation.remaining != remaining || s.ballStatus.text != text {
		t.Fatal("paused preparation changed its countdown or text")
	}
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	if s.preparation.remaining != remaining-1 {
		t.Fatal("resuming did not advance preparation exactly once")
	}
}
