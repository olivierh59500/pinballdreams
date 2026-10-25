package rules

import "github.com/olivierh59500/pinballdreams/internal/ruledata"

// applyStageEffect dispatches native game actions with extracted parameters.
// No source bytes are inspected when a bumper, target or feature is collected.
func (r *Engine) applyStageEffect(stage Stage) {
	effect := stage.Effect
	switch effect.Kind {
	case ruledata.NoEffect:
	case ruledata.SwitchStages:
		r.stages[effect.Clear].Players &^= r.mask()
		r.stages[effect.Arm].Players |= r.mask()
	case ruledata.AnnounceAdvance:
		r.announce(effect.Messages[r.actorFeatures().Advances], stage.Priority)
	case ruledata.CopyLeadingScore:
		r.copyLeadingScore()
	case ruledata.ExtraBall:
		r.grantExtraBall()
	case ruledata.SetMultiplier:
		r.BonusMultiplier = effect.Value
	case ruledata.HoldBonus:
		r.HoldBonus = true
	case ruledata.ProgressFeature:
		r.progressFeature(effect)
	case ruledata.RaiseJackpot:
		bank := r.defs.Jackpots[effect.Bank]
		r.jackpots[effect.Bank] = (r.jackpots[effect.Bank] + bank.Increment) % 1_000_000_000_000
	case ruledata.CollectJackpot:
		if label := effect.CopyText; label != nil {
			line := append([]byte(nil), r.messageText(label.Destination)...)
			copy(line[:label.Characters], r.messageText(label.Source)[:label.Characters])
			r.textOverrides[label.Destination] = line
		}
		r.collectedJackpots[effect.Bank] = r.jackpots[effect.Bank]
		r.award(r.jackpots[effect.Bank], 0)
		r.jackpots[effect.Bank] = r.defs.Jackpots[effect.Bank].Initial
	case ruledata.SavedAward:
		r.award(r.savedAwards[effect.Bank], 0)
	case ruledata.FixedAward:
		r.award(effect.Amount, 0)
	case ruledata.CopyScore, ruledata.CopyBonus:
		if r.Banks != nil {
			score, bonus := r.Banks()
			if effect.Kind == ruledata.CopyScore {
				r.award(score, 0)
			} else {
				r.award(0, bonus)
			}
		}
	case ruledata.ArmLocks:
		r.armLocks()
	case ruledata.ReserveBall:
		r.reserveBall()
	case ruledata.CollectLockedJackpot:
		r.collectLockedJackpot()
	case ruledata.ChangeGate:
		r.applyGate(r.defs.GateActions[effect.Gate])
	default:
		r.Unsupported[stage.Hook] = true
	}
}

func (r *Engine) grantExtraBall() {
	r.grantRelaunchCredit()
	if r.OnExtraBall != nil {
		r.OnExtraBall()
	}
}

func (r *Engine) progressFeature(effect ruledata.Effect) {
	if effect.Program != 0 {
		progress := r.counter(effect.Program)
		progress[r.Player] = int(byte(progress[r.Player] + 1))
		step := r.defs.Paths[effect.Program].Steps[progress[r.Player]]
		if effect.ClampProgress && !step.Selectable {
			progress[r.Player] = (progress[r.Player] - 1) & 255
		}
	}
	if group := r.group(effect.Group); group != nil {
		for _, id := range group.Lamps {
			lamp := r.Lamps[id]
			if lamp.Players&r.mask() == 0 {
				if lamp.Flags&lampExcludedFromBank == 0 {
					lamp.Players |= r.mask()
				}
				break
			}
		}
	}
	if effect.Timer != 0 {
		r.timers[effect.Timer] = effect.Value
	}
}

// copyLeadingScore retains the supplied game's comparison rule: its two
// decimal chunks are compared independently, starting with the current bank.
func (r *Engine) copyLeadingScore() {
	if r.BanksFor == nil {
		return
	}
	threshold, _ := r.BanksFor(r.Player)
	selected := 0
	for player := 1; player < r.PlayerCount; player++ {
		score, _ := r.BanksFor(player)
		if threshold/100_000_000 > score/100_000_000 || threshold%100_000_000 > score%100_000_000 {
			continue
		}
		threshold, selected = score, player
	}
	score, _ := r.BanksFor(selected)
	r.award(score, 0)
}
