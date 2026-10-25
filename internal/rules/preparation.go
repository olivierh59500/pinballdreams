package rules

// PreparationTick advances the presentation services used before live spring
// play. It leaves ball-driven features, bank collectors and queued prizes idle.
func (r *Engine) PreparationTick() {
	r.Frame++
	r.Animate(false)
	r.advanceArtwork(false, r.tilted)
	r.advanceLampTimers()
	r.sequential()
}

// ScoreCycleTick uses the source's attract lamp modes while game-over text
// owns the panel. Timers and sequential lamps run without live collectors.
func (r *Engine) ScoreCycleTick() {
	r.Frame++
	r.Animate(true)
	r.advanceArtwork(true, r.tilted)
	r.advanceLampTimers()
	r.sequential()
}
