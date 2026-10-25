package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func waitForPreparedBall(t *testing.T, s *Session) {
	t.Helper()
	for frames := 0; s.PreparingBall() && frames < 1000; frames++ {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.PreparingBall() {
		t.Fatal("supplied ball preparation did not finish")
	}
}

// This exercises the reported launch failure through complete native frames,
// including the spring detector, gravity and release ordering on every table.
func TestLaunchLeavesShooterLane(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(1)
			waitForPreparedBall(t, s)
			for range 55 {
				if err = s.Update(Input{Launch: true}); err != nil {
					t.Fatal(err)
				}
			}
			if !s.InChute {
				t.Fatalf("ball left the spring before launch: %+v", s.Engine.State)
			}
			if s.Charge != 4096 {
				t.Fatalf("full pull = %d; want 4096", s.Charge)
			}
			if err = s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
			if s.Engine.State.VY >= -3000 {
				t.Fatalf("release did not propel the ball: %+v", s.Engine.State)
			}
			minY := s.Engine.State.PixelY()
			for range 120 {
				if err = s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
				minY = min(minY, s.Engine.State.PixelY())
			}
			if minY > 300 {
				t.Fatalf("launched ball never entered the playfield: y=%d, state=%+v", minY, s.Engine.State)
			}
		})
	}
}

func TestPauseRetainsBallState(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	waitForPreparedBall(t, s)
	if err = s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	before := s.Engine.State
	for range 100 {
		if err = s.Update(Input{Left: true, Right: true, Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Engine.State != before {
		t.Fatal("paused session advanced ball or flippers")
	}
}

func TestDrainCountsBonusBeforeAdvancingPlayer(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(2)
			waitForPreparedBall(t, s)
			// Seed a drain from live play, rather than the restricted spring loop.
			s.launching = false
			s.Players[0].Bonus = 1234
			s.Rules.BonusMultiplier = 3
			s.Rules.Players[0].Boosters = 7
			s.Engine.State.Y = 520 << 10
			if err = s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
			if !s.pendingDrain || s.Bonus != nil || s.Player != 0 {
				t.Fatal("drain did not leave bonus entry to the following state")
			}
			if err = s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
			if s.Bonus == nil || s.Player != 0 || s.Players[0].Score != 0 {
				t.Fatal("drain skipped bonus presentation")
			}
			before := s.Engine.State
			for range 49 {
				if err = s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if s.Player != 0 || s.Engine.State != before {
				t.Fatal("ball or player advanced during bonus presentation")
			}
			for frame := 0; frame < 2000 && s.Bonus != nil; frame++ {
				if err = s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if s.Player != 1 || s.Ball != 1 || s.Players[0].Score != 3702 || !s.pendingNewBall {
				t.Fatalf("bonus transition: player=%d ball=%d banks=%+v", s.Player, s.Ball, s.Players[0])
			}
			waitForPreparedBall(t, s)
			if s.Players[0].Bonus != 0 {
				t.Fatal("the next ball initializer did not clear the completed bonus bank")
			}
		})
	}
}

func TestTiltSkipsOrdinaryBonusAndDefersBankReset(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(2)
	waitForPreparedBall(t, s)
	s.launching = false
	s.Players[0].Bonus = 1234
	s.Engine.Tilted = true
	s.Engine.State.Y = 520 << 10
	if err = s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if !s.pendingDrain || s.Player != 0 {
		t.Fatal("tilted drain did not retain the transition frame")
	}
	if err = s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.Bonus == nil || !s.Bonus.MovingBackdrop() || s.Player != 0 {
		t.Fatal("tilt skipped the end-ball entry hold")
	}
	for n := 0; s.Bonus != nil && n < 100; n++ {
		if err = s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Bonus != nil || s.Players[0].Score != 0 || s.Players[0].Bonus != 1234 || s.Player != 1 {
		t.Fatal("tilted ball counted points or discarded its bank before initialization")
	}
	waitForPreparedBall(t, s)
	if s.Players[0].Bonus != 0 || s.Players[0].Score != 0 {
		t.Fatal("the next ball did not clear the unheld bonus without scoring it")
	}
}

func TestSteelWheelLaunchOpensOneWayGate(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(1); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	waitForPreparedBall(t, s)
	for range 55 {
		if err = s.Update(Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	minX := s.Engine.State.PixelX()
	for range 240 {
		if err = s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
		minX = min(minX, s.Engine.State.PixelX())
	}
	// Before sensor-controlled gates were connected to their collision
	// rasters, a full launch rebounded at x=281 and returned to the spring.
	if minX >= 200 {
		t.Fatalf("shot did not pass the original upper gate: minimum x=%d", minX)
	}
}

func TestPlayMusicWaitsForOriginalLaunchSensor(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(1)
			waitForPreparedBall(t, s)
			for range 30 {
				if err = s.Update(Input{Launch: true}); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
			if !s.awaitingPlayMusic || !s.launching {
				t.Fatal("spring release started the main music before the source gate")
			}
			found := false
			for layer, definition := range s.Table.Layers {
				for _, sensor := range definition.Sensors {
					if sensor.Address != s.playMusicSensor {
						continue
					}
					found = true
					s.Engine.SetLayer(layer)
					s.Engine.State.X = int32(((sensor.X0+sensor.X1)/2 - 8) << 10)
					s.Engine.State.Y = int32(((sensor.Y0+sensor.Y1)/2 - 8) << 10)
					s.Engine.State.VX, s.Engine.State.VY, s.Engine.State.Spin = 0, 0, 0
					if err = s.Update(Input{}); err != nil {
						t.Fatal(err)
					}
					if s.awaitingPlayMusic || s.launching {
						t.Fatalf("source gate %x did not start play music", sensor.Address)
					}
					break
				}
				if found {
					break
				}
			}
			if !found {
				t.Fatal("decoded launch music has no source sensor")
			}
			if err = s.Update(Input{Music: true}); err != nil {
				t.Fatal(err)
			}
			if !s.MusicDisabled || s.Muted {
				t.Fatal("music toggle changed the all-audio mute instead of the source song selection")
			}
		})
	}
}
