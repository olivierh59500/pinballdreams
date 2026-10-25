package physics

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestRampGateChangesTheCollisionAtlas(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 1)
	if err != nil {
		t.Fatal(err)
	}
	e := New(table)
	// This is the original Steel Wheel left-ramp entry rectangle.
	e.State.X, e.State.Y = (25-8)<<10, (130-8)<<10
	e.SelectLayer()
	if e.Layer != 1 {
		t.Fatal("ramp entry retained the base collision level")
	}
	found := false
	for i, upper := range table.Layers[1].Collision[:320*400] {
		if upper == table.Layers[0].Collision[i] {
			continue
		}
		found = true
		if got := e.MaterialAt(i%320, i/320); got != upper {
			t.Fatalf("upper collision byte=%d; want %d", got, upper)
		}
		break
	}
	if !found {
		t.Fatal("source ramp atlas does not differ from the base atlas")
	}
	e.State.X, e.State.Y = (160-8)<<10, (185-8)<<10
	e.SelectLayer()
	if e.Layer != 0 {
		t.Fatal("ramp exit did not restore the base collision level")
	}
}

func TestGateChangesSurviveAVisitToTheOtherLevel(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 1)
	if err != nil {
		t.Fatal(err)
	}
	e := New(table)
	e.SetTarget(0xce3c, true)
	opened := append([]byte(nil), e.DynamicCollision...)
	if bytes.Equal(opened, table.Layers[0].Collision) {
		t.Fatal("gate did not change the base atlas")
	}
	e.SetLayer(1)
	if e.DynamicCollision != nil {
		t.Fatal("base gate raster leaked into the upper level")
	}
	e.SetLayer(0)
	if !bytes.Equal(e.DynamicCollision, opened) {
		t.Fatal("returning from a ramp discarded the open gate")
	}
}
