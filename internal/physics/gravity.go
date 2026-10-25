package physics

import "github.com/olivierh59500/pinballdreams/internal/resources"

const gravityCellWidth = 8

// SampleGravity considers the center cell, then its left and right neighbors.
// An unobstructed cell chooses a supplied force vector. Blocked cells and an
// invalid selector retain the current force for this sample. They also release
// the override, so the next unlocked frame starts with the table's default.
func (e *Engine) SampleGravity() {
	e.gravitySelected = false
	x := e.State.PixelX() + e.Table.GravityOffset
	y := e.State.PixelY() + e.Table.GravityOffset
	cell := x / gravityCellWidth
	if y < 0 || y >= len(e.collision())/320 || cell < 0 || cell >= 40 {
		return
	}
	for _, neighbor := range [3]int{cell, cell - 1, cell + 1} {
		selector, clear := e.gravityCell(neighbor, y)
		if !clear {
			continue
		}
		if selector > 0 && int(selector) <= len(e.Table.Gravity) {
			force := e.Table.Gravity[selector-1]
			e.GravityX, e.GravityY = force[0], force[1]
			e.gravitySelected = true
		}
		return
	}
}

func (e *Engine) gravityCell(cell, y int) (selector byte, clear bool) {
	if cell < 0 || cell >= 40 {
		edge := y * 2
		if cell >= 40 {
			edge++
		}
		probe := e.Table.Layers[e.Layer].GravityEdges[edge]
		if e.spanValue(probe.Clearance[0])|e.spanValue(probe.Clearance[1]) != 0 {
			return 0, false
		}
		return e.spanValue(probe.Selection), true
	}
	for x := range gravityCellWidth {
		material := e.MaterialAt(cell*gravityCellWidth+x, y)
		if material&3 != 0 {
			return 0, false
		}
		if material&4 != 0 {
			selector |= 1 << uint(7-x)
		}
	}
	return selector, true
}

func (e *Engine) spanValue(span resources.MaterialSpan) byte {
	if !span.Dynamic {
		return span.Constant
	}
	var value byte
	for x := range gravityCellWidth {
		if e.MaterialAt(span.X+x, span.Y)&span.Mask != 0 {
			value |= 1 << uint(7-x)
		}
	}
	return value
}
