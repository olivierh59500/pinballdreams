package rules

// initializeGates mirrors the source new-ball initializer's conditional gate
// service after the current player's feature-stage latches have been reset.
func (r *Engine) initializeGates() {
	for _, action := range r.defs.GateActions {
		if len(action.Conditions) > 0 {
			r.applyGate(action)
		}
	}
}

func (r *Engine) applyGate(v gateAction) {
	players := v.Players
	if len(v.Conditions) > 0 {
		players = 255
		for _, id := range v.Conditions {
			if r.stages[id].Players&r.mask() != 0 {
				players = 0
			}
		}
	}
	r.collected[v.Target] = players
	r.paintTarget(v.Target)
	if r.OnTargetLayerChange != nil {
		r.OnTargetLayerChange(v.Layer, v.Target, players&r.mask() != 0)
	}
}
