package rules

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

// Source records may be discarded after loading. Gameplay must consume only
// native rules and artwork, including captures, progression and announcements.
func TestRulesRunWithoutOriginalExecutableBytes(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			table, err := resources.LoadFS(assets.Files(), id)
			if err != nil {
				t.Fatal(err)
			}
			table.Data, table.Layers = nil, nil
			r, err := New(table)
			if err != nil {
				t.Fatal(err)
			}
			r.OnCapture = func(int, int, int16, int16) {}
			for level, layer := range r.defs.Layers {
				for _, list := range [][]ruledata.Sensor{layer.Sensors, layer.ExtraSensors} {
					for _, sensor := range list {
						r.NewBall(0)
						r.Layer = level
						r.Sensors((sensor.X0+sensor.X1)/2, (sensor.Y0+sensor.Y1)/2)
						for range 5 {
							r.Accelerator()
							r.Tick(false, false, false)
						}
					}
				}
			}
			for id, message := range r.defs.Announcements {
				r.NewBall(0)
				r.announce(id, 1)
				if r.announcement == nil {
					t.Fatal("native announcement required source bytes")
				}
				for frame := 0; frame < 600 && r.announcement != nil; frame++ {
					r.Tick(false, false, false)
				}
				if len(r.defs.Texts[message.Text]) == 0 {
					t.Fatal("announcement lost its extracted text")
				}
			}
			if frame := r.Art(false, false); frame.Bounds() != table.Art.Bounds() {
				t.Fatal("native lamp composition changed table bounds")
			}
		})
	}
}

func TestMissingNativeRulesHasNoExecutableFallback(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	table.Rules = nil
	if _, err := New(table); err == nil {
		t.Fatal("missing native definitions fell back to original program bytes")
	}
}

func TestSavedAwardKeepsItsArmingValue(t *testing.T) {
	for id := 1; id < 4; id++ {
		table, err := resources.LoadFS(assets.Files(), id)
		if err != nil {
			t.Fatal(err)
		}
		r, err := New(table)
		if err != nil {
			t.Fatal(err)
		}
		r.NewBall(0)
		var arming Stage
		var saved Stage
		for _, stage := range r.defs.Stages {
			if stage.SaveAwardOnArm != 0 {
				arming = stage
			}
			if stage.Effect.Kind == ruledata.SavedAward {
				saved = stage
			}
		}
		if arming.ID == 0 || saved.ID == 0 {
			t.Fatal("supplied saved-jackpot feature is missing")
		}
		var score uint64
		r.OnAward = func(points, bonus uint64) { score += points }
		bank := arming.SaveAwardOnArm
		r.awardSources[bank] = 12_345_678
		r.queueStage(ruledata.StageEntry{Stage: arming.ID, Arm: true}, true)
		r.awardSources[bank] = 87_654_321
		score = 0
		r.applyStageEffect(saved)
		if score != 12_345_678 {
			t.Fatalf("table %d saved jackpot=%d; want its value when armed", id+1, score)
		}
	}
}
