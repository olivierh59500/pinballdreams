package rules

import "github.com/olivierh59500/pinballdreams/internal/ruledata"

const (
	stagePolled         = 1
	stageCollectedTimer = 2
	stageArmedTimer     = 4
)

func (r *Engine) counter(id int) *[8]int {
	if c := r.counters[id]; c != nil {
		return c
	}
	c := new([8]int)
	r.counters[id] = c
	return c
}

func (r *Engine) feature(id int) { r.invoke(r.defs.Features[id]) }

// invoke advances an extracted feature path. Every transition has already
// been resolved; the game handles eligibility and player progress directly.
func (r *Engine) invoke(id int) {
	path, ok := r.defs.Paths[id]
	if !ok {
		return
	}
	if path.Mode == ruledata.Together {
		for _, entry := range path.Entries {
			r.queueFeature(entry, true)
		}
		return
	}
	progress := r.counter(id)
	index := progress[r.Player] & 255
	progress[r.Player] = index
	step, ok := path.Steps[index]
	if !ok {
		return
	}
	progress[r.Player] = step.Index
	if step.Stop {
		return
	}
	if step.Path != 0 {
		progress[r.Player] = step.Next
		r.invoke(step.Path)
	} else if r.queueStage(step.StageEntry, false) {
		progress[r.Player] = step.Next
	} else {
		progress[r.Player] = step.Unavailable
	}
}

func (r *Engine) queueFeature(entry ruledata.StageEntry, together bool) {
	if entry.Path != 0 {
		r.invoke(entry.Path)
	} else {
		r.queueStage(entry, together)
	}
}

func (r *Engine) queueStage(entry ruledata.StageEntry, together bool) bool {
	stage, ok := r.defs.Stages[entry.Stage]
	if !ok {
		return false
	}
	state := r.stages[entry.Stage]
	if entry.Arm {
		state.Players |= r.mask()
		if stage.SaveAwardOnArm != 0 {
			bank := stage.SaveAwardOnArm
			r.savedAwards[bank] = r.awardSources[bank]
		}
	} else {
		if stage.Kind != ruledata.Repeatable && state.Players&r.mask() == 0 {
			return false
		}
		consume := stage.ConsumeInPath
		if together {
			consume = stage.ConsumeInGroup
		}
		if stage.Kind == ruledata.Collectible && consume {
			state.Players &^= r.mask()
		}
	}
	r.queue = append(r.queue, action{ID: entry.Stage, Arm: entry.Arm})
	return true
}

// process starts one queued award or advances its ongoing presentation.
// Entry points and sounds run once, while a polled effect keeps its own state.
func (r *Engine) process() {
	if r.sequence != nil {
		r.stepSequence()
		return
	}
	if len(r.queue) == 0 {
		return
	}
	entry := r.queue[len(r.queue)-1]
	r.queue = r.queue[:len(r.queue)-1]
	stage := r.defs.Stages[entry.ID]
	if entry.Arm {
		r.sound(stage.OffSound)
		r.setLamp(stage.OffLamp, true)
		r.announce(stage.OffMessage, stage.OffPriority)
		if stage.Flags&stageArmedTimer != 0 {
			r.stages[stage.ID].Timer = stage.Duration
		}
		return
	}
	r.award(stage.Points, stage.Bonus)
	if stage.ID == r.defs.BoosterStage {
		r.actorFeatures().Boosters = (r.actorFeatures().Boosters + 1) & 65535
	}
	r.sound(stage.OnSound)
	if stage.ID == r.defs.AdvanceStage {
		player := r.actorFeatures()
		if player.Advances >= r.defs.AdvanceLimit {
			r.invoke(r.defs.AdvanceOverflow)
			player.Advances = 0
		} else {
			player.Advances++
		}
	}
	r.setLamp(stage.OnLamp, true)
	r.setLamp(stage.OffLamp, false)
	if stage.Flags&stagePolled != 0 {
		r.sequence = &sequence{Stage: stage.ID}
		r.stepSequence()
		return
	}
	if stage.Flags&stageCollectedTimer != 0 {
		r.stages[stage.ID].Timer = stage.Duration
	}
	r.applyStageEffect(stage)
	r.announce(stage.OnMessage, stage.Priority)
}

func (r *Engine) announce(id, priority int) {
	definition, ok := r.defs.Announcements[id]
	if !ok || priority < r.messagePriority {
		return
	}
	text := r.messageText(definition.Text)
	previous := r.message
	if r.PanelText != nil {
		previous = r.PanelText()
	}
	r.announcement = newAnnouncement(definition, text, r.messagePhase, previous)
	r.announcement.Buffer = r.messageBuffer
	r.messagePriority = priority
}
