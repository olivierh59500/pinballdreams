package resources

import "image"

// VisibleBallIndex returns one supplied ball pixel after table-cover masking.
func (t *Table) VisibleBallIndex(layer, x, y, nudge, bx, by int) byte {
	index := t.Ball.ColorIndexAt(bx, by)
	if index == 0 || t.BallHiddenOnLayer(layer, x+1+bx, y+1+nudge+by) {
		return 0
	}
	return index
}

// RolloverContact tests the visible ball against classified table colors.
// Position is the physical ball origin. Nudge offsets the table and its cover
// mask while the sprite remains at its ordinary screen position.
func (t *Table) RolloverContact(art *image.Paletted, layer, x, y, nudge int) bool {
	ballX, ballY := x+1, y+1+nudge
	for by := range t.Ball.Bounds().Dy() {
		for bx := range t.Ball.Bounds().Dx() {
			if t.VisibleBallIndex(layer, x, y, nudge, bx, by) == 0 {
				continue
			}
			px, py := ballX+bx, ballY+by
			if !image.Pt(px, py).In(art.Bounds()) {
				continue
			}
			if t.RolloverColors[art.ColorIndexAt(px, py)] {
				return true
			}
		}
	}
	return false
}
