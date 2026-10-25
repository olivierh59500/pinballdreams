package rules

// BeginQuit clears the transient flashing and collection bytes in the linked
// lamp groups. Shared blink phases, clocks and target geometry survive.
func (r *Engine) BeginQuit() {
	for _, group := range r.defs.Groups {
		for _, id := range group.Lamps {
			lamp := r.Lamps[id]
			lamp.Flags &^= lampFlash
			lamp.Players = 0
		}
	}
}
