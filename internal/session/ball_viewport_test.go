package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/physics"
)

func TestIgnitionBallAtUpperViewportBoundary(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	// The controlled extra-ball mission reaches this pose at clock 1328.
	// The ball must preserve the separator and its black neighbors while
	// retaining its visible part below the table boundary.
	s.Engine.State = physics.State{X: 276480, Y: 159804}
	s.Camera = 164
	s.prepareBallSprite()
	s.spritesHidden = true
	without := s.Frame(s.Table.Viewport.Height)
	s.spritesHidden = false
	with := s.Frame(s.Table.Viewport.Height)
	visible := 0
	for y := s.Table.Viewport.PanelHeight; y < with.Rect.Max.Y; y++ {
		for x := range with.Rect.Max.X {
			if y < s.Table.Viewport.TableTop {
				if got := with.RGBAAt(x, y); got != without.RGBAAt(x, y) {
					t.Fatalf("ball painted separator pixel %d,%d: %v", x, y, got)
				}
			} else if with.RGBAAt(x, y) != without.RGBAAt(x, y) {
				visible++
			}
		}
	}
	if visible == 0 {
		t.Fatal("partly visible ball disappeared from the table")
	}
}
