package sound

// BeginQuitFade fades the current replay while preserving its terminal-song
// policy. The original exit clears a table flag, not the song's stop latch.
func (p *Player) BeginQuitFade(target, step byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fadeTarget, p.fadeStep = target, step
}

// FadeComplete observes the master level used for the last output declaration.
// The replay advances its fade afterward, so its internal level reaches the
// target one tick before the original exit service can observe completion.
func (p *Player) FadeComplete() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.presentedMaster == p.fadeTarget
}

// PlaybackStopped reports the replay's terminal latch, separately from output
// muting. Development lifecycle traces use it to reach an earned song stop.
func (p *Player) PlaybackStopped() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopped
}
