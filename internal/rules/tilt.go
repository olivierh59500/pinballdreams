package rules

// BeginTilt applies the source entry reset without changing score, bonus,
// multiplier, awarded relaunch credits, physical target geometry or ongoing
// feature records. The source clears only the actor's booster/advance counters.
func (r *Engine) BeginTilt() {
	r.tilted = true
	r.resetBallLamps()
	player := r.actorFeatures()
	player.Boosters, player.Advances = 0, 0
	r.Locked = false
}
