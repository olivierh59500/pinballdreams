package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"testing"
)

func TestOriginalFlipperAlternativesRemainIndependent(t *testing.T) {
	for _, pair := range [][2]ebiten.Key{{ebiten.KeyShiftLeft, ebiten.KeyShiftRight}, {ebiten.KeyAltLeft, ebiten.KeyAltRight}, {ebiten.KeyMetaLeft, ebiten.KeyMetaRight}} {
		down := map[ebiten.Key]bool{pair[0]: true}
		pressed := func(key ebiten.Key) bool { return down[key] }
		if left, right := flipperKeys(pressed); !left || right {
			t.Fatalf("left key %v moved wrong flipper", pair[0])
		}
		delete(down, pair[0])
		down[pair[1]] = true
		if left, right := flipperKeys(pressed); left || !right {
			t.Fatalf("right key %v moved wrong flipper", pair[1])
		}
		down[pair[0]] = true
		if left, right := flipperKeys(pressed); !left || !right {
			t.Fatal("simultaneous original keys lost a flipper")
		}
	}
}
