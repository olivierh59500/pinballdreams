package rules

// DrawSpring applies the original power strip to the shared table surface.
// Comparing pixels first avoids cloning a published surface for an unchanged
// level, while still repainting if another table effect touched this area.
func (r *Engine) DrawSpring(level int) {
	gauge := &r.defs.Table.SpringGauge
	level = max(0, min(len(gauge.Frames)-1, level))
	if len(gauge.Frames) == 0 {
		return
	}
	if r.artwork == nil {
		r.initializeArtwork()
	}
	frame := gauge.Frames[level]
	a := r.artwork
	for y := range frame.Rect.Dy() {
		for x := range frame.Rect.Dx() {
			at := (gauge.Y+y)*a.frame.Stride + gauge.X + x
			if a.frame.Pix[at] == frame.Pix[y*frame.Stride+x] {
				continue
			}
			a.mutable()
			for row := range frame.Rect.Dy() {
				start := (gauge.Y+row)*a.frame.Stride + gauge.X
				copy(a.frame.Pix[start:start+frame.Rect.Dx()], frame.Pix[row*frame.Stride:row*frame.Stride+frame.Rect.Dx()])
			}
			return
		}
	}
}
