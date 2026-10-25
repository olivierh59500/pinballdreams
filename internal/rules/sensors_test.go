package rules

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

func TestPassageScanRetainsOriginalVerticalStop(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 2)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	// This lies inside a later Beat Box passage, but the source's ordered
	// lower-Y test has already stopped the scan at an earlier rectangle.
	r.Sensors(237, 95)
	if r.hits[0xcc6a] {
		t.Fatal("a passage beyond the source's vertical stop became active")
	}
}

func TestVerticalExitKeepsOnlyTheLaterTablesCurrentLatch(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			table, err := resources.LoadFS(assets.Files(), id)
			if err != nil {
				t.Fatal(err)
			}
			r, err := New(table)
			if err != nil {
				t.Fatal(err)
			}
			sensor := r.defs.Layers[0].Sensors[0]
			if id == 2 {
				// Beat Box's first entry starts at zero. Use its spring entry
				// to test a positive lower boundary without capture effects.
				sensors := r.defs.Layers[0].Sensors
				sensor = sensors[len(sensors)-1]
			}
			x := (sensor.X0 + sensor.X1) / 2
			r.SpringSensorContacts(x, sensor.Y0, false)
			if !r.SensorActive(sensor.Object) {
				t.Fatal("passage entry did not set its latch")
			}
			r.SpringSensorContacts(x, sensor.Y0-1, false)
			if r.SensorActive(sensor.Object) != (id != 0) {
				t.Fatal("vertical exit did not retain the source table's latch behavior")
			}
			r.SpringSensorContacts(-20, -20, false)
			if r.SensorActive(sensor.Object) {
				t.Fatal("horizontal exit did not clear the retained latch")
			}
		})
	}
}

func TestAboveTableCenterUsesUnsignedPassageCoordinates(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 2)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	sensor := r.defs.Layers[0].Sensors[0]
	x := (sensor.X0 + sensor.X1) / 2
	r.SpringSensorContacts(x, 0, false)
	if !r.SensorActive(sensor.Object) {
		t.Fatal("zero-row passage did not become active")
	}
	r.SpringSensorContacts(x, -1, false)
	if r.SensorActive(sensor.Object) {
		t.Fatal("negative center incorrectly retained a latch at the lower-Y stop")
	}
}

func TestSpringRolloverLeavesOrdinaryAwardsAndCollectionIdle(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	r.NewBall(0)
	var points uint64
	r.OnAward = func(score, bonus uint64) { points += score }
	r.SpringSensorContacts(167, 77, true)
	r.PreparationTick()
	if points != 0 {
		t.Fatal("spring scan dispatched a live-game rollover award")
	}
	for _, id := range r.defs.SkillLamps {
		if r.Lamps[id].Players&r.mask() != 0 {
			t.Fatal("spring scan collected a live-game rollover lamp")
		}
	}
	r.Sensors(-20, -20)
	r.Sensors(167, 77)
	if points != 10000 {
		t.Fatal("live scan did not dispatch the supplied rollover award")
	}
}

func TestCaptureReleaseCueWaitsForReservedBallsAndPlaysOnce(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	r.capture = &capture{Role: ruledata.TrapObject}
	r.Locked, r.locks.Active = true, true
	var cues []audiodata.Command
	r.OnSound = func(cue audiodata.Command) { cues = append(cues, cue) }
	r.releaseCapture()
	if !r.Locked || r.capture == nil || len(cues) != 0 {
		t.Fatal("an active reserved-ball state released the capture")
	}
	r.locks.Active = false
	r.releaseCapture()
	r.releaseCapture()
	if r.Locked || r.capture != nil || len(cues) != 1 {
		t.Fatalf("capture release did not publish exactly one cue: locked=%t capture=%+v cues=%v", r.Locked, r.capture, cues)
	}
	// The independent source trace declares this Ignition release note.
	if cues[0].Action != audiodata.Sound || cues[0].Note.Sample != 23 || cues[0].Note.Pitch != 20 {
		t.Fatalf("release note differs from the supplied original: %+v", cues[0])
	}
}
