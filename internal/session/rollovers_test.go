package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
)

func TestCoveredBallCannotCollectARollover(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	waitForPreparedBall(t, s)
	// The controlled WARP contact belongs to live play, after the exit gate.
	s.launching = false
	// Five restoration frames select Ignition's second WARP lane for the
	// skill shot: its 10000-point rollover also queues the 20000-point prize.
	// Hold the physical pose to isolate visible-ball contact from movement.
	s.Engine.State.X, s.Engine.State.Y = (167-8)<<10, (77-8)<<10
	s.Engine.Locked = true
	cover := s.Table.Layers[0].Hidden
	s.Table.Layers[0].Hidden = make([]byte, len(cover))
	for n := range s.Table.Layers[0].Hidden {
		s.Table.Layers[0].Hidden[n] = 1
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.Players[0].Score != 0 {
		t.Fatalf("a fully covered ball collected the rollover: %+v", s.Players[0])
	}
	s.Table.Layers[0].Hidden = cover
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.Players[0].Score != 30000 {
		t.Fatalf("the visible ball did not collect its supplied lane reward: %+v", s.Players[0])
	}
}
