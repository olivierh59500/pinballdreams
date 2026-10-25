package rules

import (
	"fmt"

	"github.com/olivierh59500/pinballdreams/internal/highscore"
)

// UpdateRankingTemplates follows the full table reset's shared ranking copy.
// Live idle announcements and attract instructions reference the same text.
func (r *Engine) UpdateRankingTemplates(scores [4]highscore.Entry) {
	for _, message := range r.defs.Table.Attract.Roaming {
		if message.ScoreRank >= 0 {
			r.textOverrides[message.Definition.Text] = message.RankingText(scores[message.ScoreRank])
		}
	}
}

// PanelState exposes display ownership without exposing source layout.
type PanelState struct {
	Blocked, Announcing, Direct bool
	Message                     string
}

type PanelSelection struct {
	Before                    PanelState
	Special, Wrote, ResetIdle bool
	Text                      string
}

func (r *Engine) messageText(id int) []byte {
	if text := r.textOverrides[id]; text != nil {
		return text
	}
	return r.defs.Texts[id]
}

// AdvancePanelSelection executes the original early live display service.
// Saved-value presentation precedes ordinary announcement selection.
func (r *Engine) AdvancePanelSelection() PanelSelection {
	r.expireTimedStages()
	result := PanelSelection{Before: r.PanelState()}
	if result.Before.Blocked {
		return result
	}
	special := r.defs.Panel.Special
	if state := r.stages[special.Stage]; special.Stage != 0 && state != nil && state.Players&r.mask() != 0 && !r.Locked {
		value := r.savedAwards[special.Bank]
		if special.Add {
			value = (value + special.Delta) % 1_000_000_000_000
			result.ResetIdle = true
		} else {
			value = (value + 1_000_000_000_000 - special.Delta) % 1_000_000_000_000
		}
		r.savedAwards[special.Bank] = value
		if special.ClearAtLowZero && value%100_000_000 == 0 {
			state.Players &^= r.mask()
		}
		if r.messagePriority < special.Priority {
			r.announcement, r.message, r.messagePriority = nil, "", 0
			result.Special = true
			if r.Frame&special.ClockMask == 0 {
				result.Wrote, result.Text = true, r.rawText(special.Text)
			}
			return result
		}
	}
	r.advanceAnnouncement()
	return result
}

// SeedPanelSpecial sets the controlled saved-value display fixture.
func (r *Engine) SeedPanelSpecial(value uint64) {
	special := r.defs.Panel.Special
	if state := r.stages[special.Stage]; state != nil {
		state.Players |= r.mask()
		r.savedAwards[special.Bank] = value
	}
}

func (r *Engine) PanelState() PanelState {
	return PanelState{Blocked: r.sequence != nil, Announcing: r.announcement != nil, Direct: r.directMessage, Message: r.message}
}

// PanelAnimation shares the native announcement sequencer with idle programs.
type PanelAnimation struct {
	engine     *Engine
	state      *announcement
	sharedText bool
}

func (r *Engine) NewPanelAnimation(id int) *PanelAnimation {
	definition, ok := r.defs.Announcements[id]
	if !ok {
		return nil
	}
	previous := r.message
	if r.PanelText != nil {
		previous = r.PanelText()
	}
	text := r.messageText(definition.Text)
	state := newAnnouncement(definition, text, r.messagePhase, previous)
	state.Buffer = r.messageBuffer
	return &PanelAnimation{engine: r, state: state, sharedText: true}
}

func (p *PanelAnimation) Step() (text string, wrote, done bool) {
	if p.sharedText {
		p.state.Text = p.engine.messageText(p.state.TextID)
	}
	text, wrote = p.state.step()
	p.engine.messagePhase, p.engine.messageBuffer = p.state.Phase, p.state.Buffer
	return text, wrote, p.state.Done
}

// UpdatePanelTemplates follows the late live-frame status-template service.
// Idle display programs therefore observe these values on their next call.
func (r *Engine) UpdatePanelTemplates(ball int) {
	for _, binding := range r.defs.Panel.Bindings {
		var value uint64
		switch binding.Kind {
		case "player":
			value = uint64(r.Player + 1)
		case "ball":
			value = uint64(ball)
		case "multiplier":
			value = uint64(r.BonusMultiplier)
		case "jackpot":
			value = r.jackpots[binding.Bank]
		case "collected-jackpot":
			value = r.collectedJackpots[binding.Bank]
		case "award-source":
			value = r.awardSources[binding.Bank]
		case "saved-award":
			value = r.savedAwards[binding.Bank]
		case "advance-tens":
			// The supplied helper reads the first byte of a four-digit bank
			// and writes its high nibble twice, retaining the source's quirk.
			value = uint64(r.actorFeatures().Advances/1000%10) * 11
		}
		power := uint64(1)
		for range binding.Digits {
			power *= 10
		}
		text := fmt.Sprintf("%*d", binding.Digits, value%power)
		if binding.ZeroPad {
			text = fmt.Sprintf("%0*d", binding.Digits, value%power)
		}
		line := r.textOverrides[binding.Text]
		if line == nil {
			line = append([]byte(nil), r.defs.Texts[binding.Text]...)
		}
		copy(line[binding.Position:binding.Position+binding.Digits], text)
		r.textOverrides[binding.Text] = line
	}
}
