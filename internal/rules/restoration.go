package rules

// TargetRestoration keeps the source bank/list order while restoring a ball's
// targets. Stationary contacts clear their latch without consuming a frame.
type TargetRestoration struct {
	engine  *Engine
	targets []int
	cursor  int
}

func (r *Engine) BeginTargetRestoration(player int) *TargetRestoration {
	// Player masks switch before the common serve resets current-ball features.
	r.Player = player
	var targets []int
	for _, group := range r.defs.TargetGroups {
		for _, id := range group.Targets {
			targets = append(targets, id)
			kind := r.defs.Targets[id].Kind
			if kind != 1 && kind != 8 {
				// The source leaves this bank after its first stationary contact.
				break
			}
		}
	}
	return &TargetRestoration{engine: r, targets: targets}
}

// Pending clears contact latches up to the next movable target. The current
// target's latch clears before its preparation frame, as in the source loop.
func (t *TargetRestoration) Pending() bool {
	r := t.engine
	for t.cursor < len(t.targets) {
		id := t.targets[t.cursor]
		r.collected[id] &^= r.mask()
		kind := r.defs.Targets[id].Kind
		if kind == 1 || kind == 8 {
			return true
		}
		t.cursor++
	}
	return false
}

// Restore runs after the frame's presentation service. It publishes one target
// raster and one supplied reset cue, then advances to the next ordered record.
func (t *TargetRestoration) Restore() int {
	r := t.engine
	if t.cursor >= len(t.targets) {
		return 0
	}
	id := t.targets[t.cursor]
	t.cursor++
	r.paintTarget(id)
	if r.OnTargetChange != nil {
		r.OnTargetChange(id, false)
	}
	r.sound(r.defs.TargetCycle.ResetSound)
	return id
}

// Finish initializes per-ball features after the last target has returned.
func (t *TargetRestoration) Finish() { t.engine.initializeBall() }
