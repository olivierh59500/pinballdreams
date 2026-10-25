package rules

import (
	"fmt"

	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

type rouletteState struct {
	Started             bool
	Index, Delay, Speed int
}
type sequence struct {
	Stage    int
	Roulette rouletteState
	Counter  *BonusCounter
}

func (r *Engine) rawText(id int) string {
	text := r.defs.Texts[id]
	if override := r.textOverrides[id]; override != nil {
		text = override
	}
	if len(text) > 0 && text[len(text)-1] == 255 {
		text = text[:len(text)-1]
	}
	return string(text)
}

func (r *Engine) directText(text string) {
	var line [20]byte
	for i := range line {
		line[i] = ' '
	}
	if r.PanelText != nil {
		copy(line[:], r.PanelText())
	}
	copy(line[:], text)
	r.message = string(line[:])
	r.directMessage = true
}

// stepSequence polls the same collected stage until its native callback finishes.
// Its base award and entry sound have already been applied exactly once.
func (r *Engine) stepSequence() {
	s := r.sequence
	stage := r.defs.Stages[s.Stage]
	effect := stage.Effect
	done := false
	if def := r.defs.Sequences.Roulettes[effect.Value]; effect.Kind == ruledata.SelectRoulette && len(def.Entries) > 0 {
		done = r.stepRoulette(def, &s.Roulette)
	} else if text := effect.Text; effect.Kind == ruledata.CountBonus {
		if s.Counter == nil {
			amount := uint64(0)
			if r.Banks != nil {
				_, amount = r.Banks()
			}
			s.Counter = r.newCounter(amount)
			r.directText(r.rawText(text))
		} else {
			prefix := r.rawText(text)
			if len(prefix) > 10 {
				prefix = prefix[:10]
			}
			r.directText(prefix + fmt.Sprintf("%10d", s.Counter.Remaining%10_000_000_000))
			r.panelNumber(s.Counter.Remaining)
			done = s.Counter.advance()
		}
	} else if effect.Kind == ruledata.ReserveBall {
		r.reserveBall()
		done = true
	} else {
		// The supplied Ignition/Beat Box roulette lists are empty. Preserve
		// that evidence boundary instead of supplying invented prize records.
		r.Unsupported[stage.Hook] = true
		done = true
	}
	if done {
		r.sequence = nil
		if r.announcement != nil {
			r.announcement.New = false
		}
	}
}

func (r *Engine) stepRoulette(def rouletteDefinition, s *rouletteState) bool {
	if !s.Started {
		s.Started = true
		s.Index = int(def.Lookup[r.Frame&127])
		s.Delay, s.Speed = def.InitialSpeed>>2, def.InitialSpeed
		r.sound(def.Sound)
		return false
	}
	for _, entry := range def.Entries {
		r.setLamp(entry.Lamp, false)
	}
	if s.Index == len(def.Entries) {
		s.Index = 0
	}
	if s.Index < 0 || s.Index >= len(def.Entries) {
		r.Unsupported[def.Hook] = true
		return true
	}
	entry := def.Entries[s.Index]
	r.setLamp(entry.Lamp, true)
	text := entry.Text
	if text == 0 {
		text = def.DefaultText
	}
	r.directText(r.rawText(text))
	s.Delay--
	if s.Delay < 0 {
		s.Speed++
		if s.Speed == def.Limit {
			s.Started = false
			r.invoke(entry.Program)
			return true
		}
		s.Delay = s.Speed >> 2
		s.Index++
	}
	return false
}
