package physics

func clampVelocity(value int32) int16 {
	return int16(min(int32(velocityLimit), max(int32(-velocityLimit), value)))
}

// Resolve computes a rebound in the contact's normal/tangent coordinates.
// Integer divisions and explicit narrowings preserve the supplied tuning's
// quantization, including the transfer between sliding motion and ball spin.
// The result reports whether a powered impact remains armed for gameplay.
func (e *Engine) Resolve(contact Contact) bool {
	ball := &e.State
	if contact.Material < 0 || contact.Material >= len(e.Table.Materials) {
		return false
	}
	material := e.Table.Materials[contact.Material]
	if material.TangentDrag == 0 || material.SpinDrag == 0 || material.BounceDivisor == 0 {
		return false
	}
	relativeX := clampVelocity(int32(ball.VX) + int32(contact.SurfaceX))
	relativeY := clampVelocity(int32(ball.VY) + int32(contact.SurfaceY) + int32(ball.NudgeSpeed))
	inverseAngle := (angleCount - contact.Angle) & (angleCount - 1)
	sine, cosine := int32(e.Table.Sine[inverseAngle]), int32(e.Table.Cosine[inverseAngle])
	// The response uses four times the velocity precision while resolving
	// friction. The inverse projection below returns to integration units.
	normal := int16((int32(relativeX)*cosine - int32(relativeY)*sine) >> 13)
	tangent := int16((int32(relativeY)*cosine + int32(relativeX)*sine) >> 13)
	if normal <= 0 {
		return false
	}
	normal = -normal
	slope := int16((int32(tangent) << 4) / int32(normal))
	if slope < 0 {
		slope = -slope
	}
	if uint16(slope) >= uint16(material.SlopeLimit) || uint16(normal) >= uint16(material.NormalLimit) {
		normal = 0
	}
	armed := false
	if normal != 0 && contact.Powered {
		normal, armed = kickerImpulse(normal, material.KickThreshold, material.KickImpulse)
	}
	normal -= int16((int32(normal) << 8) / int32(material.BounceDivisor))
	slidingDrag, spinDrag := material.TangentDrag, material.SpinDrag
	if normal >= -1023 {
		scale := (-normal >> 6) + 1
		slidingDrag = int16(uint16(slidingDrag) * uint16(scale))
		spinDrag = int16(uint16(spinDrag) * uint16(scale))
	}
	slip := int32(int16(ball.Spin+ball.NudgeSpeed-tangent)) << 8
	ball.Spin -= int16(slip / int32(spinDrag))
	tangent += int16(slip / int32(slidingDrag))
	tangent = int16(int32(tangent) * 2048 / 2049)
	angle := contact.Angle & (angleCount - 1)
	sine, cosine = int32(e.Table.Sine[angle]), int32(e.Table.Cosine[angle])
	worldX := int16((int32(normal)*cosine - int32(tangent)*sine) >> 15)
	worldY := int16((int32(tangent)*cosine + int32(normal)*sine) >> 15)
	ball.VX = clampVelocity(int32(worldX) - int32(contact.SurfaceX))
	ball.VY = clampVelocity(int32(worldY) - int32(contact.SurfaceY) - int32(ball.NudgeSpeed))
	if contact.Depth >= 6 {
		ball.X += (-256 * cosine) >> 14
		ball.Y += (-256 * sine) >> 14
	}
	return armed
}

// Slingshots require a sufficiently direct impact; other powered contacts use
// the stronger bumper kick. A qualifying response keeps the impact armed.
func kickerImpulse(normal, threshold, impulse int16) (int16, bool) {
	if int32(normal) <= -int32(threshold) {
		return normal - impulse, true
	}
	return normal, false
}
