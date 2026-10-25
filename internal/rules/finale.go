package rules

// HeldBonusFinale runs the original final-ball retained-bonus announcement and
// one additional multiplied count. It does not grant invented match prizes.
type HeldBonusFinale struct {
	engine       *Engine
	announcement *announcement
	counter      *BonusCounter
	amount       uint64
	text         string
	done         bool
}

// BeginHeldBonusFinale starts after ordinary last-ball counting and boosters.
// Its amount is the retained bonus bank at that boundary, including boosters.
func (r *Engine) BeginHeldBonusFinale(amount uint64) *HeldBonusFinale {
	definition := r.defs.Announcements[r.defs.Bonus.FinalHoldMessage]
	previous := r.message
	if r.PanelText != nil {
		previous = r.PanelText()
	}
	line := newAnnouncement(definition, r.defs.Texts[definition.Text], r.messagePhase, previous)
	line.Buffer = r.messageBuffer
	r.sound(r.defs.Bonus.FinalHoldSound)
	return &HeldBonusFinale{engine: r, announcement: line, amount: amount, text: previous}
}

// Step follows one original wait boundary. The announcement has no table
// presentation service; the subsequent count uses the ordinary lamp service.
func (f *HeldBonusFinale) Step() bool {
	if f.done {
		return true
	}
	if f.announcement != nil {
		f.engine.BeginLiveFrame()
		text, wrote := f.announcement.step()
		if wrote {
			f.text = text
		}
		f.engine.messagePhase = f.announcement.Phase
		f.engine.messageBuffer = f.announcement.Buffer
		if f.announcement.Done {
			f.announcement = nil
			f.counter = f.engine.newCounter(f.amount)
			f.counter.text = f.text
			f.counter.targetMultiplier = max(1, f.engine.BonusMultiplier)
			f.counter.writeBonusPrefix()
			if f.counter.targetMultiplier > 1 {
				f.counter.startMultiplierBeat()
			}
			f.text = f.counter.Message()
		}
		return false
	}
	f.counter.Step()
	f.text = f.counter.Message()
	// The source bonus-only helper returns as soon as its digit service ends;
	// this second count has no booster pass or post-count/exit hold.
	f.done = f.counter.phase == bonusAfterCount
	return f.done
}

func (f *HeldBonusFinale) Message() string { return f.text }
