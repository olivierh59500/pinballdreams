// Package frontend implements the opening and table selection in native Go.
// It consumes extracted pictures, glyphs, text and timing, without inspecting
// executable records or retaining the original program's state layout.
package frontend

import "github.com/olivierh59500/pinballdreams/internal/resources"

// fade moves each color component by one artwork intensity level per frame.
// Completion includes a settled frame, matching the supplied presentation.
func fade(colors, target []resources.Ink) (settled bool) {
	settled = true
	for n, goal := range target {
		current := colors[n]
		colors[n] = resources.Ink{R: approach(current.R, goal.R), G: approach(current.G, goal.G), B: approach(current.B, goal.B)}
		if current != goal {
			settled = false
		}
	}
	return settled
}

func approach(value, goal uint8) uint8 {
	if value < goal {
		return value + 1
	}
	if value > goal {
		return value - 1
	}
	return value
}
