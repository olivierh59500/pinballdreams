package rules

// BeginLiveFrame advances the shared game clock before motion and contacts.
// Feature choices made by passages therefore see the current frame's clock.
func (r *Engine) BeginLiveFrame() { r.Frame++ }

// BeforeContacts expires timed stages and advances an existing announcement.
// Announcements created by this frame's events start on the following frame.
func (r *Engine) BeforeContacts() {
	r.expireTimedStages()
	r.advanceAnnouncement()
}

// FinishLiveFrame runs the live table services after both contact award banks.
// Newly collected banks enter the queue after this frame's action service,
// while presentation observes the effects already applied by that service.
func (r *Engine) FinishLiveFrame(rotate bool) {
	r.advanceSkill()
	r.process()
	r.Animate(false)
	r.advanceArtwork(false, false)
	r.advanceLampTimers()
	r.sequential()
	r.collectTargetBanks()
	r.completeGroups()
	r.restoreTargets()
	r.flash()
	if rotate {
		r.rotate()
	}
	r.decrementTargetCooldowns()
}

// FinishCapture follows the session's late nudge/tilt service. Camera travel
// must finish before an otherwise completed trap presentation releases motion.
func (r *Engine) FinishCapture(cameraTransition bool) {
	r.releaseCaptureAfterCamera(cameraTransition)
}

// FinishTiltFrame advances only the presentation and target cooldown services
// of the tilted-ball loop. Its caller owns the already-advanced frame clock.
func (r *Engine) FinishTiltFrame() {
	r.Animate(false)
	r.advanceArtwork(false, true)
	r.advanceLampTimers()
	r.sequential()
	r.decrementTargetCooldowns()
}

func (r *Engine) advanceSkill() {
	if !r.skill {
		return
	}
	for _, id := range r.defs.SkillLamps {
		lamp := r.Lamps[id]
		lamp.Phase = !lamp.Phase
		// The source tests the complete collection byte here, including bits
		// retained for other players, before ending the active skill sequence.
		if lamp.Players == 0 {
			continue
		}
		if lamp.Flags&lampFlash != 0 {
			if r.defs.SkillPath != 0 {
				r.invoke(r.defs.SkillPath)
			} else {
				r.feature(r.defs.SkillGroup)
			}
		}
		for _, other := range r.defs.SkillLamps {
			r.Lamps[other].Flags &^= lampFlash
			r.Lamps[other].Phase = false
		}
		r.skill = false
		break
	}
}
