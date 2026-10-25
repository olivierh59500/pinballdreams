package rules

import "fmt"

// BonusCounter retains the source's decimal-digit count-up: each nonzero digit
// contributes its place value, then the service advances to the next digit.
type BonusCounter struct {
	r                          *Engine
	original, Remaining, place uint64
	Multiplier                 int
	counter, digits            int
	done                       bool
	displayed                  uint64
	boosters, boosterDisplayed int
	boosterShown, finished     bool
	skipCounting               bool
	phase                      bonusPhase
	hold, targetMultiplier     int
	text                       string
}

type bonusPhase int

const (
	bonusCounting bonusPhase = iota
	bonusEntry
	bonusMultiplying
	bonusAfterCount
	bonusBoosterWait
	bonusExit
)

// BeginBonus starts the original multiplier presentation before counting points.
func (r *Engine) BeginBonus(amount uint64) *BonusCounter {
	if !r.tilted {
		// A normal loss ends the active reservation service. Occupied slot
		// records survive until ordinary serving resets the current ball.
		r.locks.Active = false
	}
	b := r.newCounter(amount)
	b.boosters = r.actorFeatures().Boosters
	b.targetMultiplier = max(1, r.BonusMultiplier)
	b.phase, b.hold = bonusEntry, r.defs.Bonus.EntryFrames
	b.text = r.defs.Bonus.EntryText
	r.resetBallLamps()
	r.RefreshRelaunchIndicator()
	for _, sound := range r.defs.Bonus.EntrySounds {
		r.sound(sound)
	}
	return b
}

// BeginTiltedBonus keeps the entry hold, lamps and cues, then omits ordinary
// counting and boosters. A final held bonus remains eligible for its encore.
func (r *Engine) BeginTiltedBonus(amount uint64) *BonusCounter {
	r.tilted = true
	b := r.BeginBonus(amount)
	b.skipCounting = true
	return b
}

// MovingBackdrop identifies the source's initial frozen-motion presentation.
// Flipper/nudge return animation continues during this hold, without ball motion.
func (b *BonusCounter) MovingBackdrop() bool { return b.phase == bonusEntry }

func (r *Engine) newCounter(amount uint64) *BonusCounter {
	return &BonusCounter{r: r, original: amount, Remaining: amount, displayed: amount, place: 1, Multiplier: 1, counter: r.defs.Bonus.InitialDelay, digits: 15}
}

// Step finishes after score counting and the following retained-bonus updates.
func (b *BonusCounter) Step() bool {
	if b.finished {
		return true
	}
	b.r.PreparationTick()
	switch b.phase {
	case bonusEntry:
		b.hold--
		if b.hold == 0 {
			if b.skipCounting {
				b.finished = true
				return true
			}
			b.writeBonusPrefix()
			if b.targetMultiplier > 1 {
				b.startMultiplierBeat()
			} else {
				b.phase = bonusCounting
			}
		}
	case bonusMultiplying:
		b.writeRemaining()
		b.hold--
		if b.hold == 0 {
			if b.Multiplier < b.targetMultiplier {
				b.Multiplier++
				b.Remaining = (b.Remaining + b.original) % 1_000_000_000_000
				b.r.scoreActivity()
				b.writeBonusPrefix()
				b.startMultiplierBeat()
			} else {
				b.phase = bonusCounting
			}
		}
	case bonusCounting:
		b.writeRemaining()
		if b.advance() {
			b.phase, b.hold = bonusAfterCount, b.r.defs.Bonus.AfterCountDelay
		}
	case bonusAfterCount:
		b.hold--
		if b.hold == 0 {
			b.phase, b.hold = bonusBoosterWait, b.r.defs.Bonus.BoosterPeriod
		}
	case bonusBoosterWait:
		b.hold--
		if b.hold == 0 {
			if b.boosters == 0 {
				b.phase, b.hold = bonusExit, b.r.defs.Bonus.ExitDelay
			} else {
				b.boosterShown, b.boosterDisplayed = true, b.boosters
				b.writeBooster()
				b.boosters--
				b.r.addToBanks(0, b.r.defs.BoosterValue)
				b.r.sound(b.r.defs.Bonus.BoosterSound)
				b.hold = b.r.defs.Bonus.BoosterPeriod
			}
		}
	case bonusExit:
		b.hold--
		if b.hold == 0 {
			b.finished = true
		}
	}
	return b.finished
}

func (b *BonusCounter) startMultiplierBeat() {
	b.phase, b.hold = bonusMultiplying, b.r.defs.Bonus.MultiplierDelay
	b.r.sound(b.r.defs.Bonus.MultiplierSound)
}

func (b *BonusCounter) writeBonusPrefix() {
	var cells [20]byte
	for i := range cells {
		cells[i] = ' '
	}
	copy(cells[:], b.text)
	copy(cells[:], b.r.defs.Bonus.Text)
	cells[9] = byte('0' + b.Multiplier)
	b.text = string(cells[:])
}

func (b *BonusCounter) writeRemaining() {
	b.displayed = b.Remaining
	b.r.panelNumber(b.displayed)
	b.writeBonusPrefix()
	b.text = b.text[:10] + fmt.Sprintf("%10d", b.displayed%10_000_000_000)
}

func (b *BonusCounter) writeBooster() {
	line := []byte(fmt.Sprintf("%-20s", b.r.defs.Bonus.BoosterText))
	position := b.r.defs.Bonus.BoosterPosition
	copy(line[position-2:position], fmt.Sprintf("%2d", b.boosterDisplayed%100))
	b.text = string(line[:20])
}

// advance services only the decimal bank. Polled table features share the
// normal frame clock instead of advancing the end-ball presentation clock.
func (b *BonusCounter) advance() bool {
	if b.done {
		return true
	}
	b.counter--
	if b.counter != 0 {
		return false
	}
	b.counter = b.r.defs.Bonus.Period
	if b.Remaining/b.place%10 != 0 {
		b.Remaining -= b.place
		b.r.addToBanks(b.place, 0)
		b.r.sound(b.r.defs.Bonus.CountSound)
		return false
	}
	b.digits--
	if b.digits == 0 {
		b.done = true
		return true
	}
	b.place *= 10
	if b.Remaining == 0 {
		b.done = true
		return true
	}
	return false
}

// Message uses the supplied bonus label and source multiplier character slot.
func (b *BonusCounter) Message() string {
	if b.text != "" {
		return b.text
	}
	if b.boosterShown {
		line := []byte(fmt.Sprintf("%-20s", b.r.defs.Bonus.BoosterText))
		position := b.r.defs.Bonus.BoosterPosition
		copy(line[position-2:position], fmt.Sprintf("%2d", b.boosterDisplayed%100))
		return string(line[:20])
	}
	line := []byte(b.r.defs.Bonus.Text)
	line[9] = byte('0' + b.Multiplier)
	return string(line) + fmt.Sprintf("%10d", b.displayed%10_000_000_000)
}
