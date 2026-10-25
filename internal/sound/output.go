package sound

// SampleValueClock reports the renderer's value-duration profile for independent
// recording comparisons. It does not advance or configure playback.
func SampleValueClock() int64 { return pitchClock }

// OutputDeclaration describes native PCM ranges without hardware addresses or
// register state. Development traces may observe changes without moving replay.
type OutputDeclaration struct {
	Start, Length, LoopStart, LoopLength int
	Period, Volume                       uint16
	Starts                               uint32
	Enabled                              bool
}

// OutputDeclarations copies the current declarations without advancing notes,
// sample positions or output filters. Changes to the copy cannot affect audio.
func (p *Player) OutputDeclarations() [4]OutputDeclaration {
	p.mu.Lock()
	defer p.mu.Unlock()
	var result [4]OutputDeclaration
	for index, out := range p.streams {
		result[index] = OutputDeclaration{
			Start: out.Start, Length: out.Length,
			LoopStart: out.LoopStart, LoopLength: out.LoopLength,
			Period: out.Period, Volume: out.Volume, Starts: out.Starts, Enabled: out.Enabled,
		}
	}
	return result
}
