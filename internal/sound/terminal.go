package sound

// PrepareGameOver bounds the following loop tails without restarting the
// current sample segments or changing the tracker's own loop definitions.
func (p *Player) PrepareGameOver(loopBytes int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.streams {
		p.streams[i].LoopLength = loopBytes
	}
}

// StopAtReturn keeps an end-of-ball song from returning to ordinary play music.
// The source sets this latch after its entry command pair has been applied.
func (p *Player) StopAtReturn() {
	p.mu.Lock()
	p.returnStop = true
	p.mu.Unlock()
}
