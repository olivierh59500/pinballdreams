package session

import (
	"image"

	"github.com/olivierh59500/pinballdreams/internal/physics"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

// paintFlippers follows the original two-plane copy. Source frame buffers
// already contain the original low-plane backdrop; live upper planes retain
// table lighting and EHB shading instead of reverting to a static image.
func paintFlippers(frame *image.RGBA, art *image.Paletted, table *resources.Table, engine *physics.Engine, camera, nudge, tableTop int) {
	for _, flipper := range table.Flippers {
		phase := engine.VisualFlipperFrame(flipper.Right)
		phase = max(0, min(16, phase))
		for y := range 43 {
			worldY := flipper.Y + y
			screenY := worldY - camera - nudge + tableTop
			if screenY < frame.Rect.Min.Y || screenY >= frame.Rect.Max.Y {
				continue
			}
			for x := range 64 {
				worldX := flipper.X + x
				if worldX < frame.Rect.Min.X || worldX >= frame.Rect.Max.X {
					continue
				}
				index := art.ColorIndexAt(worldX, worldY)&^3 | flipper.Art[phase].ColorIndexAt(x, y)&3
				frame.Set(worldX, screenY, art.Palette[index])
			}
		}
	}
}
