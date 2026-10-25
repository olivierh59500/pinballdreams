package sound

// BeginTilt starts the supplied one-shot song and stops at its return point.
// The source sets that stop latch after selecting the temporary song.
func (p *Player) BeginTilt(command Command) {
	p.mu.Lock()
	p.selectMusic(command.Order, command.Repeats, byte(command.Priority))
	p.returnStop = true
	observer := p.commandObserver
	p.mu.Unlock()
	if observer != nil {
		observer(command)
	}
}
