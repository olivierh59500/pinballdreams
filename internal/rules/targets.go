package rules

// Targets identifies the contacted source polygon through its original
// bounding collections, then updates its native latch, light and rule program.
func (r *Engine) Targets(x, y, material int, powered bool) {
	if material == 5 || powered {
		return
	}
	for _, group := range r.defs.TargetGroups {
		if group.Layers&(1<<uint(r.Layer)) == 0 {
			continue
		}
		if x < group.X0 || x > group.X1 || y < group.Y0 || y > group.Y1 {
			continue
		}
		for _, id := range group.Targets {
			target := r.defs.Targets[id]
			if x < target.X0 || x > target.X1 || y < target.Y0 || y > target.Y1 {
				continue
			}
			if r.collected[id]&r.mask() != 0 || target.Kind == 7 && r.targetCooldown[id] > 0 {
				return
			}
			if target.Kind == 1 || target.Kind == 8 {
				r.collected[id] |= r.mask()
				r.paintTarget(id)
				if r.OnTargetChange != nil {
					r.OnTargetChange(id, true)
				}
			}
			if target.Kind == 7 {
				r.targetCooldown[id] = r.defs.TargetCycle.RetriggerFrames
			}
			r.setLamp(target.Lamp, true)
			r.event(group.ID, false)
			return
		}
	}
}

func (r *Engine) completeTargets() {
	r.collectTargetBanks()
	r.restoreTargets()
}

// collectTargetBanks queues a completed bank before the lamp-bank collector.
// Its geometry returns later in the same frame's separate restoration service.
func (r *Engine) collectTargetBanks() {
	if len(r.resetTargets) == 0 {
		for _, group := range r.defs.TargetGroups {
			if group.Layers&(1<<uint(r.Layer)) == 0 {
				continue
			}
			if len(group.Targets) == 0 {
				continue
			}
			complete := true
			for _, id := range group.Targets {
				if r.collected[id]&r.mask() == 0 {
					complete = false
					break
				}
			}
			if !complete {
				continue
			}
			r.feature(group.ID)
			r.resetTargets = append([]int(nil), group.Targets...)
			r.resetTargetTimer = r.defs.TargetCycle.ResetDelay
			break
		}
	}
}

func (r *Engine) restoreTargets() {
	if len(r.resetTargets) == 0 {
		return
	}
	r.resetTargetTimer--
	if r.resetTargetTimer > 0 {
		return
	}
	id := r.resetTargets[0]
	r.resetTargets = r.resetTargets[1:]
	r.resetTargetTimer = r.defs.TargetCycle.ResetPeriod
	if len(r.resetTargets) == 0 {
		r.resetTargetTimer = 0
	}
	r.collected[id] &^= r.mask()
	r.paintTarget(id)
	if r.OnTargetChange != nil {
		r.OnTargetChange(id, false)
	}
	r.sound(r.defs.TargetCycle.ResetSound)
}
