package game

import "github.com/hajimehoshi/ebiten/v2"

var leftFlipperKeys = [...]ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyAltLeft, ebiten.KeyMetaLeft, ebiten.KeyArrowLeft, ebiten.KeyZ}
var rightFlipperKeys = [...]ebiten.Key{ebiten.KeyShiftRight, ebiten.KeyAltRight, ebiten.KeyMetaRight, ebiten.KeyArrowRight, ebiten.KeySlash}

// flipperKeys retains the original Shift/Alt/Amiga alternatives, with Meta
// standing for the host's Amiga-equivalent key, plus the native convenience keys.
func flipperKeys(down func(ebiten.Key) bool) (left, right bool) {
	for _, key := range leftFlipperKeys {
		left = left || down(key)
	}
	for _, key := range rightFlipperKeys {
		right = right || down(key)
	}
	return left, right
}
