package physics

// MaterialAt samples table materials with each flipper's selected silhouette
// replacing its own collision channel in the base playfield.
func (e *Engine) MaterialAt(x, y int) byte {
	if x < 0 || x >= 320 || y < 0 || y >= len(e.collision())/320 {
		return 0
	}
	collision := e.collision()
	material := collision[y*320+x]
	// Prepared flipper frames own their whole channel, including the static
	// walls composed during extraction. Copying them also replaces any later
	// changes to that channel within the swept rectangle.
	if e.Layer != 0 {
		return material
	}
	for n := range e.Table.Flippers {
		f := &e.Table.Flippers[n]
		fx, fy := x-f.X, y-f.Y
		if fx < 0 || fx >= 64 || fy < 0 || fy >= 43 {
			continue
		}
		phase := e.State.LeftFrame()
		if f.Right {
			phase = e.State.RightFrame()
		}
		phase = min(16, max(0, phase))
		channel := byte(1)
		if f.Right {
			channel = 2
		}
		material &^= channel
		if f.Mask[phase][fy*64+fx] {
			material |= channel
		}
	}
	return material
}

// Probe averages the exact source angles of all intersecting perimeter points.
// Unwrapping across angle zero is selected by the original quadrant bit masks.
func (e *Engine) Probe() (Contact, bool) {
	x, y := e.State.PixelX(), e.State.PixelY()+e.State.Nudge()
	count, sum, quadrants, lower, last := 0, 0, 0, 0, 0
	// The last matching probe in the source's unrolled order selects material.
	for _, i := range e.Table.ProbeOrder {
		p := e.Table.Points[i]
		if e.MaterialAt(x+p.X, y+p.Y)&3 == 0 {
			continue
		}
		count++
		sum += p.Angle
		quadrants |= 1 << uint(i/11)
		if i < 22 {
			lower++
		}
		last = i
	}
	if count == 0 {
		return Contact{}, false
	}
	if quadrants == 9 || quadrants == 11 || quadrants == 13 {
		sum += lower * 2048
	}
	angle := sum / count & 2047
	materialPoint := e.Table.Points[last]
	material := int(e.MaterialAt(x+materialPoint.X, y+materialPoint.Y))
	contact := Contact{Angle: angle, Depth: count, Material: material}
	// Choose the nearest perimeter point from the resolved contact angle.
	probe := (angle*len(e.Table.Points) + angleCount/2) / angleCount
	if probe >= len(e.Table.Points) {
		probe -= len(e.Table.Points)
	}
	px, py := x+e.Table.Points[probe].X, e.State.PixelY()+e.Table.Points[probe].Y
	contact.PointX, contact.PointY = px, py
	if material == 1 || material == 2 {
		for n := range e.Table.Flippers {
			f := &e.Table.Flippers[n]
			if f.Right != (material == 2) {
				continue
			}
			dx, dy := int16(px-f.PivotX), int16(py-f.PivotY)
			if (!f.Right && dx < 0) || (f.Right && dx >= 0) {
				break
			}
			speed := e.State.LeftSpeed
			if f.Right {
				dx, dy = -dx, -dy
				speed = e.State.RightSpeed
			}
			contact.SurfaceX = int16(int32(dy) * int32(speed))
			contact.SurfaceY = -int16(int32(dx) * int32(speed))
			break
		}
	}
	if !e.Tilted && (material == 3 || material == 7) {
		for _, k := range e.Table.Layers[e.Layer].Kickers {
			if k.Material == material && px >= k.X0 && px < k.X1 && py >= k.Y0 && py < k.Y1 {
				contact.Powered = true
				contact.Object = k.Address
				break
			}
		}
	}
	return contact, true
}
