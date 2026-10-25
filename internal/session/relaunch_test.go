package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

func TestRelaunchCreditSurvivesReadinessAndAWeakShot(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(2)
			waitForPreparedBall(t, s)
			credits := 2
			if s.Table.Rules.Relaunch.Mode == ruledata.RelaunchSingle {
				credits = 1
			}
			s.Rules.RelaunchCredits = credits
			s.finishBall()
			if s.Player != 0 || s.Ball != 1 || s.Rules.RelaunchCredits != credits {
				t.Fatal("ball-end selection consumed the relaunch credit or advanced its owner")
			}
			if err := s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
			waitForPreparedBall(t, s)
			if s.ballStatus.mode != relaunchStatus || s.Rules.RelaunchCredits != credits {
				t.Fatal("relaunch readiness discarded its credit or chose the normal status")
			}
			if err := s.Update(Input{Launch: true}); err != nil {
				t.Fatal(err)
			}
			for range 100 {
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if !s.launching || s.Rules.RelaunchCredits != credits {
				t.Fatal("a weak shot consumed a credit before the play sensor")
			}
			found := false
			for layer, definition := range s.Table.Layers {
				for _, sensor := range definition.Sensors {
					if sensor.Address != s.Table.Launch.PlaySensor {
						continue
					}
					found = true
					s.Engine.SetLayer(layer)
					s.Rules.Layer = layer
					s.Engine.State.X = int32(((sensor.X0+sensor.X1)/2 - 8) << 10)
					s.Engine.State.Y = int32(((sensor.Y0+sensor.Y1)/2 - 8) << 10)
					s.Engine.State.VX, s.Engine.State.VY, s.Engine.State.Spin = 0, 0, 0
					if err := s.Update(Input{}); err != nil {
						t.Fatal(err)
					}
					break
				}
				if found {
					break
				}
			}
			if !found || s.launching || s.Rules.RelaunchCredits != credits-1 {
				t.Fatal("play entry did not consume exactly the supplied relaunch credit")
			}
		})
	}
}
