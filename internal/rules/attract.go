package rules

import (
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

// ResetTableAttract follows source state zero's complete lamp/stage/path reset.
// It preserves shared clocks, locks, current multiplier/hold/relaunch globals,
// physical target state and the already-presented table surface.
func (r *Engine) ResetTableAttract() {
	r.Player = 0
	r.announcement = nil
	r.message, r.messagePriority, r.directMessage = "", 0, false
	r.queue = nil
	// A full table reset restores both the running jackpot and its last
	// collected display value. Linked/saved feature awards retain their banks.
	for id, bank := range r.defs.Jackpots {
		r.jackpots[id], r.collectedJackpots[id] = bank.Initial, bank.Initial
	}
	if r.artwork == nil {
		// A newly loaded source table starts from its supplied surface. State
		// zero invalidates draw caches; it does not pre-darken every lamp.
		r.initializeArtwork()
		copy(r.artwork.frame.Pix, r.defs.Table.Art.Pix)
	}
	r.ResetAttractLamps()
	if r.artwork != nil {
		for player := range r.artwork.byPlayer {
			for n := range r.artwork.byPlayer[player] {
				r.artwork.byPlayer[player][n] = true
			}
		}
	}
	r.resetTablePrograms()
	for player := range r.Players {
		r.Players[player].Boosters, r.Players[player].Advances = 0, 0
	}
}

// PrepareGame follows the source start pan's final reset before serving.
// It keeps the presented surface and invalidates the current actor's lamp
// cache, while restoring the table's declared stage/path/timer programs.
func (r *Engine) PrepareGame() {
	if !r.defs.ResetAtPanEnd {
		return
	}
	r.ResetAttractLamps()
	r.resetTablePrograms()
}

func (r *Engine) resetTablePrograms() {
	for _, id := range r.defs.Table.Attract.ResetStages {
		definition, exists := r.defs.Stages[id]
		if !exists {
			continue
		}
		state := r.stages[id]
		state.Players = 0
		if definition.Kind == ruledata.Repeatable {
			state.Players = 255
		}
		if definition.Flags&(stageCollectedTimer|stageArmedTimer) != 0 {
			state.Timer = 0
		}
	}
	for _, id := range r.defs.Table.Attract.ResetArmedStages {
		if state := r.stages[id]; state != nil {
			state.Players = 255
		}
	}
	for _, id := range r.defs.Table.Attract.ResetPaths {
		if path, exists := r.defs.Paths[id]; exists && path.Mode != ruledata.Together {
			delete(r.counters, id)
		}
	}
	for _, id := range r.defs.Table.Attract.ResetTimers {
		if stage := r.stages[id]; stage != nil {
			stage.Timer = 0
		}
	}
}

// NewAttractAnimation shares original character clocks and retained display
// bytes with gameplay, while consuming already-decoded table instructions.
func (r *Engine) NewAttractAnimation(message resources.AttractMessage) *PanelAnimation {
	previous := r.message
	if r.PanelText != nil {
		previous = r.PanelText()
	}
	state := newAnnouncement(message.Definition, message.Text, r.messagePhase, previous)
	state.Buffer = r.messageBuffer
	return &PanelAnimation{engine: r, state: state}
}

// ResetAttractLamps follows the full lamp-clear and current-player draw-cache
// initializer at the original attract phase boundary. Shared blink clocks stay.
func (r *Engine) ResetAttractLamps() {
	for _, lamp := range r.Lamps {
		lamp.Flags &^= lampFlash
		lamp.Players = 0
	}
	if r.artwork == nil {
		return
	}
	for n := range r.artwork.byPlayer[r.actorIndex()] {
		r.artwork.byPlayer[r.actorIndex()][n] = true
	}
	r.artwork.shown = r.artwork.byPlayer[r.actorIndex()]
	r.artwork.owner = r.actorIndex()
}

// AttractGuidedWait advances the full presentation service with attract
// animation temporarily disabled. Source guided mode suppresses blink mixing
// in the ordinary cached renderer during this short transition hold.
func (r *Engine) AttractGuidedWait() {
	r.Frame++
	r.Animate(false)
	r.drawAttractWait()
	r.advanceLampTimers()
	r.sequential()
}

func (r *Engine) drawAttractWait() {
	if r.artwork == nil {
		r.initializeArtwork()
	}
	a := r.artwork
	remaining := r.defs.LampDrawBudget
	for n, id := range a.lampIDs {
		lamp := r.Lamps[id]
		on := lamp.Players&r.mask() != 0
		if lamp.Flags&lampBlinkWhenCollected != 0 && on {
			on = false
		}
		if lamp.Flags&lampFlash != 0 {
			on = lamp.Phase
		}
		if r.tilted {
			on = false
		}
		if a.shown[n] == on {
			continue
		}
		a.shown[n] = on
		a.paint(r.defs.Lamps[id].Pixels, on)
		remaining--
		if remaining == 0 {
			break
		}
	}
}

// AttractGuidedMessage uses the alternate source service: advance all shared
// lamp clocks, then draw only the selected bank from its blink bytes. It does
// not consume or update the ordinary per-player draw cache.
func (r *Engine) AttractGuidedMessage(groupID int) {
	r.Frame++
	r.Animate(true)
	group := r.group(groupID)
	if group == nil {
		return
	}
	if r.artwork == nil {
		r.initializeArtwork()
	}
	for n, id := range group.Lamps {
		if n == r.defs.LampDrawBudget {
			break
		}
		r.artwork.paint(r.defs.Lamps[id].Pixels, r.Lamps[id].Blink)
	}
}
