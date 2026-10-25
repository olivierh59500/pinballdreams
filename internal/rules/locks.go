package rules

type lockState struct {
	Armed, Active, Replacement bool
	Slots                      []bool
}

// ReservedBallsActive selects the source replacement-ball status and behavior.
func (r *Engine) ReservedBallsActive() bool { return r.locks.Active }

// CaptureContact retains the source entry classification during camera travel.
func (r *Engine) CaptureContact() bool { return r.captureContact }

// ReplacementPending leaves the current live frame intact until its successor
// enters the replacement service.
func (r *Engine) ReplacementPending() bool { return r.locks.Replacement }

func (r *Engine) armLocks() {
	if r.locks.Active {
		return
	}
	r.locks.Armed = true
	for _, slot := range r.defs.Locks.Slots {
		if lamp := r.Lamps[slot.Lamp]; lamp != nil {
			lamp.Flags |= lampAnimateDuringPlay | lampBlinkWhenCollected
			lamp.Players |= r.mask()
		}
	}
}

func (r *Engine) reserveBall() {
	if !r.locks.Armed {
		return
	}
	r.locks.Active, r.locks.Replacement, r.Locked = true, true, true
	if state := r.stages[r.defs.Locks.CaptureStage]; state != nil {
		state.Players |= r.mask()
	}
}

func (r *Engine) collectLockedJackpot() {
	copies := 1
	for _, occupied := range r.locks.Slots {
		if occupied {
			copies++
		}
	}
	bank := r.defs.Locks.SuperBank
	r.collectedJackpots[bank] = r.jackpots[bank]
	r.award(uint64(copies)*r.jackpots[bank], 0)
	r.jackpots[bank] = r.defs.Jackpots[bank].Initial
	r.patchText(r.defs.Locks.SuperDigit, byte('0'+copies))
	r.clearLocks()
}

// enterLock keeps an occupied hole from awarding its feature a second time.
// New captures reserve the slot before the ordinary source feature is queued.
func (r *Engine) enterLock(object int) bool {
	if !r.locks.Armed {
		return false
	}
	for i, slot := range r.defs.Locks.Slots {
		if slot.Object != object {
			continue
		}
		if r.locks.Slots[i] {
			return true
		}
		r.locks.Slots[i] = true
		r.locks.Active = true
		if lamp := r.Lamps[slot.Lamp]; lamp != nil {
			lamp.Flags &^= lampAnimateDuringPlay | lampBlinkWhenCollected
		}
		return false
	}
	return false
}

func (r *Engine) clearLocks() {
	r.locks = lockState{Slots: make([]bool, len(r.defs.Locks.Slots))}
	for _, slot := range r.defs.Locks.Slots {
		if lamp := r.Lamps[slot.Lamp]; lamp != nil {
			lamp.Flags &^= lampAnimateDuringPlay | lampBlinkWhenCollected
			lamp.Players = 0
		}
	}
}

// TakeReplacement consumes a lock-triggered spring service without resetting
// the player's multiplier, target bank, lamps or feature-program counters.
func (r *Engine) TakeReplacement() bool {
	if !r.locks.Replacement {
		return false
	}
	r.locks.Replacement = false
	r.capture = nil
	r.sequence = nil
	r.directMessage = false
	r.Locked = false
	r.Layer = 0
	r.queue = nil
	r.announcement = nil
	r.message = ""
	r.messagePriority = 0
	return true
}

// patchText updates decoded writable slots in private message copies.
func (r *Engine) patchText(id int, value byte) {
	for _, slot := range r.defs.TextSlots[id] {
		line := r.textOverrides[slot.Text]
		if line == nil {
			line = append([]byte(nil), r.defs.Texts[slot.Text]...)
		}
		line[slot.Offset] = value
		r.textOverrides[slot.Text] = line
	}
}
