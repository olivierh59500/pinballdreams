package sound

// pitchClock is the supplied PAL note-period rate. Integer time is expressed
// on a common grid: one output frame is pitchClock units and a source-value
// duration is notePeriod*SampleRate units. No fractional drift accumulates.
const pitchClock int64 = 3546895

func (p *Player) readSample(at int) int8 {
	if at < 0 || at >= len(p.data) {
		return 0
	}
	return int8(p.data[at])
}

// mixVoice averages the source values held over one output frame. A note
// restart begins its supplied segment; loop changes take effect on segment
// completion, and pitch changes preserve the duration already in progress.
func (p *Player) mixVoice(i int) float64 {
	out, v := &p.streams[i], &p.rendered[i]
	if !out.Enabled {
		return 0
	}
	if v.generation != out.Starts {
		*v = renderedVoice{generation: out.Starts, start: out.Start, length: out.Length,
			remaining: int64(out.Period) * SampleRate, value: p.readSample(out.Start),
			period: out.Period, holding: out.Period == 0}
	} else if v.period == 0 && out.Period != 0 {
		// Restoring a running pitch schedules the next value after one source
		// clock. Zero pitch holds a value; it is not a silence command.
		v.remaining, v.holding = SampleRate, false
	}
	v.period = out.Period
	var area int64
	for left := pitchClock; left > 0; {
		if v.length <= 0 {
			break
		}
		if v.holding {
			area += int64(v.value) * left
			break
		}
		if v.remaining == 0 {
			v.index++
			if v.index == v.length {
				v.start, v.length, v.index = out.LoopStart, out.LoopLength, 0
				if v.length <= 0 {
					break
				}
			}
			v.value = p.readSample(v.start + v.index)
			if out.Period == 0 {
				v.holding = true
				continue
			}
			v.remaining = int64(out.Period) * SampleRate
		}
		duration := min(left, v.remaining)
		area += int64(v.value) * duration
		v.remaining -= duration
		left -= duration
	}
	// Two full-volume signed eight-bit voices fill one signed 16-bit side.
	return float64(area) * float64(out.Volume) * 2 / float64(pitchClock)
}
