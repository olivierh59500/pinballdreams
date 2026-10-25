package game

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

func TestMouseSpringReadsFreeVerticalMotionAndPrimaryPress(t *testing.T) {
	s := &session.Session{Table: &resources.Table{}}
	var mouse mouseSpring
	input := session.Input{Left: true}
	mouse.apply(s, &input, 100, false, false)
	if !input.Left || input.SpringMotion != 0 || input.SpringClick {
		t.Fatal("initial pointer sample moved the spring or lost a flipper")
	}
	input = session.Input{Launch: true, Right: true}
	mouse.apply(s, &input, 123, false, false)
	if !input.Launch || !input.Right || input.SpringMotion != 23 || input.SpringClick {
		t.Fatal("free pointer travel did not merge with held keyboard/flipper controls")
	}
	input = session.Input{}
	mouse.apply(s, &input, 129, true, false)
	if input.SpringMotion != 6 || !input.SpringClick {
		t.Fatal("primary press lost the current motion or did not request a shot")
	}
	input = session.Input{}
	mouse.apply(s, &input, 129, true, false)
	if !input.SpringClick || input.SpringMotion != 0 {
		t.Fatal("held primary button did not retain the original release action")
	}
	input = session.Input{}
	mouse.apply(s, &input, 129, false, false)
	if input.SpringClick || input.SpringMotion != 0 {
		t.Fatal("primary button release repeated the shooting action")
	}
}

func TestMouseSpringRetainsFractionalTravel(t *testing.T) {
	s := &session.Session{Table: &resources.Table{}}
	var mouse mouseSpring
	var input session.Input
	mouse.apply(s, &input, 10, false, false)
	for _, y := range []float64{10.25, 10.5, 10.75, 11} {
		input = session.Input{}
		mouse.apply(s, &input, y, false, false)
		if y < 11 && input.SpringMotion != 0 {
			t.Fatal("fractional travel moved early")
		}
	}
	if input.SpringMotion != 1 {
		t.Fatal("slow travel was lost between frames")
	}
	input = session.Input{}
	mouse.apply(s, &input, 10, false, false)
	if input.SpringMotion != -1 {
		t.Fatal("reverse travel changed direction or distance")
	}
}

func TestMouseSpringConsumesButtonsAcrossTransitions(t *testing.T) {
	s := &session.Session{}
	var mouse mouseSpring
	var input session.Input
	mouse.apply(s, &input, 10, true, false)
	s.Table = &resources.Table{}
	mouse.apply(s, &input, 120, true, false)
	input = session.Input{}
	mouse.apply(s, &input, 130, false, false)
	if input.SpringClick {
		t.Fatal("a button inherited from selection released the spring")
	}
	input = session.Input{}
	mouse.apply(s, &input, 140, true, false)
	if !input.SpringClick {
		t.Fatal("a fresh primary press after release was lost")
	}
	input = session.Input{}
	mouse.apply(s, &input, 160, true, true)
	mouse.apply(s, &input, 190, true, false)
	mouse.apply(s, &input, 200, false, false)
	if input.SpringClick {
		t.Fatal("a blocked or unfocused primary press crossed its release boundary")
	}
	input = session.Input{}
	mouse.apply(s, &input, 210, false, false)
	if input.SpringMotion != 10 {
		t.Fatal("free travel after the boundary was not accepted")
	}
}

func TestSuspendCancelsMouseChargeAndWaitsForBothButtons(t *testing.T) {
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	g := &Game{Session: s}
	var input session.Input
	g.mouseSpring.apply(s, &input, 50, false, false)
	g.mouseSpring.apply(s, &input, 90, false, false)
	if err := s.Update(input); err != nil || s.Charge == 0 {
		t.Fatalf("drag did not charge before suspension: %v", err)
	}
	g.Suspend()
	if s.Charge != 0 || !s.Paused || g.mouseSpring.active {
		t.Fatal("suspension retained a mouse pull")
	}
	blocked := g.controls.apply(s, session.Input{Pause: true}, nil, true, 262)
	if blocked != (session.Input{}) || !g.controls.waitForRelease {
		t.Fatal("held mouse button bypassed the release barrier")
	}
	g.controls.apply(s, session.Input{}, nil, false, 262)
	if g.controls.waitForRelease {
		t.Fatal("releasing both buttons did not free the input barrier")
	}
}

func TestVerticalMouseTravelBeyondWindowLaunchesEveryTable(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := session.New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.EnableFrameAudio(nil)
			if err = s.Update(session.Input{Start: true, Players: 1}); err != nil {
				t.Fatal(err)
			}
			for step := 0; s.PreparingBall() && step < 512; step++ {
				if err = s.Update(session.Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if s.PreparingBall() {
				t.Fatal("ball preparation did not finish")
			}
			var mouse mouseSpring
			var input session.Input
			mouse.apply(s, &input, 0, false, false)
			for y := 50; y <= 650; y += 50 {
				input = session.Input{}
				mouse.apply(s, &input, float64(y), false, false)
				if err = s.Update(input); err != nil {
					t.Fatal(err)
				}
			}
			if s.Charge < s.Table.SpringLimit-255 || !s.InChute {
				t.Fatalf("unheld vertical travel failed to pull the spring: %d", s.Charge)
			}
			input = session.Input{}
			mouse.apply(s, &input, 650, true, false)
			if !input.SpringClick {
				t.Fatal("primary press did not release the spring")
			}
			if err = s.Update(input); err != nil {
				t.Fatal(err)
			}
			if s.Charge != 0 || s.Engine.State.VY >= 0 {
				t.Fatal("primary press did not launch the ball")
			}
		})
	}
}

func TestMouseModeAndResizeBoundaries(t *testing.T) {
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	g := &Game{Session: s}
	if !g.tableMouseEnabled() {
		t.Fatal("desktop table did not enable the original mouse controller")
	}
	g.Touch = true
	if g.tableMouseEnabled() {
		t.Fatal("touch mode enabled the desktop mouse controller")
	}
	g.Touch = false
	s.Paused = true
	if g.tableMouseEnabled() {
		t.Fatal("pause kept the mouse captured")
	}
	s.Paused = false
	g.Layout(640, 480)
	var input session.Input
	g.mouseSpring.apply(s, &input, 10, false, false)
	g.Layout(1280, 960)
	g.mouseSpring.apply(s, &input, 1000, false, false)
	if input.SpringMotion != 0 || input.SpringClick {
		t.Fatal("resizing created synthetic mouse input")
	}
}
