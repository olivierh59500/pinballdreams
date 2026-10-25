package physics

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestSourceWallMaterialsAroundMovingFlippers(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	// Full original initialization composes the static wall channels into
	// each flipper frame. A partial initializer incorrectly yielded 6 and 5.
	for _, c := range []struct {
		name              string
		x, y, left, right int
		want              Contact
	}{
		{"left", 81, 434, 8, 4, Contact{PointX: 86, PointY: 449, Angle: 636, Depth: 2, Material: 7}},
		{"right", 199, 437, 12, 14, Contact{PointX: 211, PointY: 452, Angle: 325, Depth: 11, Material: 7}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := New(table)
			e.State.X, e.State.Y = int32(c.x<<10), int32(c.y<<10)
			e.State.LeftPhase, e.State.RightPhase = int16(c.left), int16(c.right)
			e.State.NudgePosition = 1024
			if got, hit := e.Probe(); !hit || got != c.want {
				t.Fatalf("native contact %+v (hit=%t); original %+v", got, hit, c.want)
			}
		})
	}
}

func TestFinalDrainFrameMatchesOriginalMotion(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	// These consecutive states come from original continuous motion traces.
	// One needs the full drain margin; the other releases a stale force field.
	for _, c := range []struct {
		name     string
		before   State
		controls Controls
		after    State
		gravity  int16
	}{
		{
			name: "contact beyond the former drain margin",
			before: State{X: 134137, Y: 521515, VX: 513, VY: 2920, Spin: 457,
				LeftPosition: 55, RightPosition: 880, RightPhase: 16},
			controls: Controls{Right: true},
			after: State{X: 136104, Y: 528990, VX: 428, VY: -1053, Spin: -221,
				LeftPosition: 55, RightPosition: 880, RightPhase: 16},
			gravity: 12,
		},
		{
			name: "default force after leaving a selected field",
			before: State{X: 121583, Y: 524419, VX: -164, VY: 917, Spin: 351,
				LeftPosition: 880, RightPosition: 880, LeftPhase: 16, RightPhase: 16},
			controls: Controls{Left: true, Right: true},
			after: State{X: 120927, Y: 528147, VX: -164, VY: 957, Spin: 347,
				LeftPosition: 880, RightPosition: 880, LeftPhase: 16, RightPhase: 16},
			gravity: 10,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := New(table)
			e.State, e.GravityY = c.before, 12
			e.SampleGravity()
			e.Step(c.controls)
			e.SelectLayer()
			e.SampleGravity()
			if e.State != c.after || e.GravityX != 0 || e.GravityY != c.gravity {
				t.Fatalf("native state %+v gravity=(%d,%d); original state %+v gravity=(0,%d)", e.State, e.GravityX, e.GravityY, c.after, c.gravity)
			}
		})
	}
}

func TestNightmareSlingshotThresholdFromContinuousMission(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 3)
	if err != nil {
		t.Fatal(err)
	}
	e := New(table)
	// Consecutive original states after the controlled reserve mission. The
	// 922-unit normal impact stays below Nightmare's 1000-unit sling threshold;
	// applying the other tables' 750-unit threshold added an incorrect kick.
	e.State = State{X: 61493, Y: 386414, VX: -382, VY: 322, Spin: 2876,
		LeftPosition: 55, RightPosition: 55}
	e.Step(Controls{})
	want := State{X: 60959, Y: 387443, VX: 366, VY: 111, Spin: 2524,
		LeftPosition: 55, RightPosition: 55}
	if e.State != want || e.ImpactObject != 0 {
		t.Fatalf("native state %+v impact%x; original %+v with no powered impact", e.State, e.ImpactObject, want)
	}
}
