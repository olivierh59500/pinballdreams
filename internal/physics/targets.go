package physics

// SetTargetOnLayer updates a specified atlas while retaining the ball's level.
// Beat Box's feature callbacks can change a gate on the other level.
func (e *Engine) SetTargetOnLayer(level, id int, active bool) {
	previous := e.Layer
	e.SetLayer(level)
	e.SetTarget(id, active)
	e.SetLayer(previous)
}

// SetTarget applies the extracted standing/active material changes to a
// private atlas. Immutable table geometry remains shared between sessions.
func (e *Engine) SetTarget(id int, active bool) {
	changes, ok := e.Table.TargetPatches[id]
	if !ok {
		return
	}
	if e.DynamicCollision == nil {
		e.DynamicCollision = append([]byte(nil), e.Table.Layers[e.Layer].Collision...)
	}
	for _, change := range changes {
		if change.Index < 0 || change.Index >= len(e.DynamicCollision) {
			continue
		}
		value := change.Standing
		if active {
			value = change.Active
		}
		e.DynamicCollision[change.Index] = e.DynamicCollision[change.Index]&^change.Mask | value
		for _, overlap := range e.Table.CollisionOverlaps {
			if overlap.FromLayer != e.Layer || change.Index < overlap.FromIndex || change.Index >= overlap.FromIndex+overlap.Length {
				continue
			}
			other := e.dynamicLayers[overlap.ToLayer]
			if other == nil {
				other = append([]byte(nil), e.Table.Layers[overlap.ToLayer].Collision...)
				e.dynamicLayers[overlap.ToLayer] = other
			}
			index := overlap.ToIndex + change.Index - overlap.FromIndex
			other[index] = other[index]&^change.Mask | value
		}
	}
}
