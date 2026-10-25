package sound

// channel owns native note, modulation and loop state. Sample locations are
// offsets into a private PCM buffer.
type channel struct {
	pitch, effect, parameter                                                uint16
	sampleStart, loopStart                                                  int
	sampleLength, loopLength, period                                        uint16
	tune, volume, direction, porta                                          byte
	target                                                                  uint16
	vibrato, vibratoPhase, tremolo, tremoloPhase                            byte
	waveforms, glissFunk, sampleOffset, loopRow, loopCount, funkAccumulator byte
	funkPosition                                                            int
}

// streamVoice describes a native PCM segment and its following loop. A new
// segment begins only on an actual source note/retrigger, not a period change.
type streamVoice struct {
	Start, Length, LoopStart, LoopLength int
	Period, Volume                       uint16
	Starts                               uint32
	Enabled                              bool
}

func (p *Player) resetSequence() {
	p.speed, p.tick = 1, 0
	p.row, p.rowCursor = 0, 0
	p.returnStop, p.stopped = false, false
}

// selectMusic preserves the original priority and return-position behavior.
// A zero priority establishes the normal song; temporary songs can return to it.
func (p *Player) selectMusic(order int, repeats, priority byte) {
	if priority == 0 {
		p.priority, p.repeats = 0, 0
		p.order, p.savedOrder = uint16(order), uint16(order)
	} else {
		if p.priority == 0 {
			p.savedOrder = p.order
		} else if priority < p.priority {
			return
		}
		p.order, p.priority, p.repeats = uint16(order), priority, repeats
	}
	p.resetSequence()
}

func (p *Player) startVoice(i int) {
	v, out := &p.channels[i], &p.streams[i]
	out.Start, out.Length = v.sampleStart, int(v.sampleLength)*2
	out.Starts++
	out.Enabled = true
}

func (p *Player) updateLoop(i int) {
	v, out := &p.channels[i], &p.streams[i]
	out.LoopStart, out.LoopLength = v.loopStart, int(v.loopLength)*2
}

func (p *Player) retrigger(i int) {
	p.startVoice(i)
	p.updateLoop(i)
	// The original retrigger publishes the loop length and current period
	// together; delayed notes therefore change pitch at this point.
	p.streams[i].Period = p.channels[i].period
}

func (p *Player) playNote(i int, n Note) {
	v, out := &p.channels[i], &p.streams[i]
	if v.pitch != 0 {
		out.Period = v.period
	}
	v.pitch, v.effect, v.parameter = uint16(n.Pitch), uint16(n.Effect), uint16(n.Parameter)
	if n.Sample > 0 && n.Sample <= len(p.module.Samples) {
		s := &p.module.Samples[n.Sample-1]
		v.sampleStart, v.sampleLength = s.Offset, uint16(s.Length/2)
		v.tune, v.volume = byte(s.Finetune), byte(s.Volume)
		v.loopStart, v.loopLength = s.Offset+s.LoopStart, uint16(s.LoopLength/2)
		v.funkPosition = v.loopStart
		if s.LoopStart != 0 || s.LoopLength != 2 {
			v.sampleLength = uint16((s.LoopStart + s.LoopLength) / 2)
		}
	}
	if v.pitch != 0 {
		if v.effect == effectExtended && v.parameter&0xf0 == 0x50 {
			v.tune = byte(v.parameter & 15)
		} else if v.effect == effectPortamento || v.effect == effectPortamentoVolume {
			v.target = p.module.Period(v.tune, int(v.pitch))
			v.direction = 0
			if int16(v.target) < int16(v.period) {
				v.direction = 1
			}
			if v.target == v.period {
				v.target = 0
			}
			p.immediate(i)
			return
		} else if v.effect == effectSampleOffset {
			// The supplied player applies 9xx before starting the segment,
			// then visits its immediate effect again afterward.
			p.immediate(i)
		}
		v.period = p.module.Period(v.tune, int(v.pitch))
		if v.effect != effectExtended || v.parameter&0xf0 != 0xd0 {
			if v.waveforms&4 == 0 {
				v.vibratoPhase = 0
			}
			if v.waveforms&64 == 0 {
				v.tremoloPhase = 0
			}
			out.Period = v.period
			p.startVoice(i)
		}
	}
	p.immediate(i)
}

func (p *Player) masterVolume(i int) {
	p.presentedMaster = p.master
	scaled := uint16(p.channels[i].volume) * uint16(p.master)
	p.streams[i].Volume = scaled / 64
	// The supplied player carries this division remainder into the next
	// voice's repeated-note cadence. It matters while the master fades.
	p.repeatVolumeRemainder = int(scaled % 64)
}

func (p *Player) immediate(i int) {
	v := &p.channels[i]
	p.updateFunk(v)
	switch v.effect {
	case effectSampleOffset:
		if v.parameter != 0 {
			v.sampleOffset = byte(v.parameter)
		}
		offset := uint16(v.sampleOffset) * 128
		if int16(offset) >= int16(v.sampleLength) {
			v.sampleLength = 1
		} else {
			v.sampleLength -= offset
			v.sampleStart += int(offset) * 2
		}
	case effectPositionJump:
		p.order = v.parameter & 127
		if p.repeats != 0 {
			p.repeats--
			if p.repeats == 0 {
				p.order, p.priority = p.savedOrder, 0
				if p.returnStop {
					p.stopped = true
					for j := range p.streams {
						p.streams[j].Volume, p.streams[j].Enabled = 0, false
					}
				}
			}
		}
		p.breakRow, p.positionJump = 0, true
	case effectSetVolume:
		v.volume = byte(min(64, v.parameter))
	case effectPatternBreak:
		// This player ignores Dxx's decimal row parameter.
		p.breakRow, p.nextOrder = 0, true
	case effectExtended:
		p.extended(i)
	case effectSpeed:
		if v.parameter != 0 {
			p.tick, p.speed = 0, byte(v.parameter)
		}
	default:
		p.streams[i].Period = v.period
	}
	p.masterVolume(i)
}

func (p *Player) updateFunk(v *channel) {
	if speed := v.glissFunk >> 4; speed != 0 {
		v.funkAccumulator += p.module.FunkRates[speed]
		if v.funkAccumulator&128 != 0 {
			v.funkAccumulator = 0
			v.funkPosition++
			if v.funkPosition >= v.loopStart+int(v.loopLength)*2 {
				v.funkPosition = v.loopStart
			}
			if at := v.funkPosition; at >= 0 && at < len(p.data) {
				p.sampleFingerprint += uint32(at+1) * uint32(int(^p.data[at])-int(p.data[at]))
				p.data[at] = ^p.data[at]
			}
		}
	}
}

func (p *Player) slide(i int, down bool) {
	v := &p.channels[i]
	delta := v.parameter & uint16(p.fineMask)
	p.fineMask = 255
	if down {
		v.period += delta
		if v.period&4095 >= 856 {
			v.period = v.period&0xf000 | 856
		}
	} else {
		v.period -= delta
		if v.period&4095 < 113 {
			v.period = v.period&0xf000 | 113
		}
	}
	p.streams[i].Period = v.period & 4095
}

func (p *Player) portamento(i int) {
	v := &p.channels[i]
	if v.target == 0 {
		return
	}
	if v.direction == 0 {
		v.period += uint16(v.porta)
		if int16(v.period) >= int16(v.target) {
			v.period, v.target = v.target, 0
		}
	} else {
		v.period -= uint16(v.porta)
		if int16(v.period) <= int16(v.target) {
			v.period, v.target = v.target, 0
		}
	}
	period := v.period
	if v.glissFunk&15 != 0 {
		for n := 0; n < 36; n++ {
			period = p.module.Period(v.tune, n)
			if v.period >= period {
				break
			}
		}
	}
	p.streams[i].Period = period
}

func rememberNibbles(previous byte, parameter uint16) byte {
	if parameter&15 != 0 {
		previous = previous&0xf0 | byte(parameter&15)
	}
	if parameter&0xf0 != 0 {
		previous = previous&15 | byte(parameter&0xf0)
	}
	return previous
}

func (p *Player) wave(phase, waveform, rampSign byte) uint16 {
	index := phase >> 2 & 31
	switch waveform & 3 {
	case 0:
		return uint16(p.module.Wave[index])
	case 1:
		if int8(rampSign) < 0 {
			return uint16(255 - index*8)
		}
		return uint16(index * 8)
	default:
		return 255
	}
}

func (p *Player) vibrato(i int) {
	v := &p.channels[i]
	delta := p.wave(v.vibratoPhase, v.waveforms, v.vibratoPhase) * uint16(v.vibrato&15) >> 7
	period := v.period
	if int8(v.vibratoPhase) >= 0 {
		period += delta
	} else {
		period -= delta
	}
	p.streams[i].Period = period
	v.vibratoPhase += v.vibrato >> 2 & 60
}

func (p *Player) volumeSlide(i int, fine bool, down bool) {
	v := &p.channels[i]
	delta := int(v.parameter & 15)
	if !fine {
		if v.parameter>>4 != 0 {
			delta, down = int(v.parameter>>4), false
		} else {
			down = true
		}
	}
	volume := int(v.volume)
	if down {
		volume = max(0, volume-delta)
	} else {
		volume = min(64, volume+delta)
	}
	v.volume = byte(volume)
	p.streams[i].Volume = uint16(v.volume)
}

func (p *Player) extended(i int) {
	v := &p.channels[i]
	value := byte(v.parameter & 15)
	switch v.parameter >> 4 {
	case 0:
		p.filter = value&1 == 0
	case 1, 2:
		if p.tick == 0 {
			p.fineMask = 15
			p.slide(i, v.parameter>>4 == 2)
		}
	case 3:
		v.glissFunk = v.glissFunk&0xf0 | value
	case 4:
		v.waveforms = v.waveforms&0xf0 | value
	case 5:
		v.tune = value
	case 6:
		if p.tick == 0 {
			if value == 0 {
				v.loopRow = byte(p.row)
			} else {
				if v.loopCount != 0 {
					v.loopCount--
					if v.loopCount == 0 {
						break
					}
				} else {
					v.loopCount = value
				}
				p.breakRow, p.patternLoop = v.loopRow, true
			}
		}
	case 7:
		v.waveforms = v.waveforms&15 | value<<4
	case 9:
		if value == 0 {
			break
		}
		due := p.tick == 0 && v.pitch == 0
		if p.tick != 0 {
			phase, enabled := int(p.retriggerPhase[value]), true
			if remainder := p.repeatVolumeRemainder; remainder >= 0 {
				// The original cadence has a finite quotient range. A volume
				// remainder at least as large as the interval suppresses the
				// repeat; smaller ones contribute their inherited phase.
				enabled = remainder < int(value)
				phase = remainder * 65536 % int(value)
			}
			due = enabled && (int(p.tick)+phase)%int(value) == 0
		}
		if due {
			p.retrigger(i)
		}
	case 10, 11:
		if p.tick == 0 {
			p.volumeSlide(i, true, v.parameter>>4 == 11)
		}
	case 12:
		if p.tick == value {
			v.volume, p.streams[i].Volume = 0, 0
		}
	case 13:
		if p.tick == value && v.pitch != 0 {
			p.retrigger(i)
		}
	case 14:
		if p.tick == 0 && p.delay == 0 {
			p.pendingDelay = value + 1
		}
	case 15:
		if p.tick == 0 {
			v.glissFunk = v.glissFunk&15 | value<<4
			if value != 0 {
				p.updateFunk(v)
			}
		}
	}
}

func (p *Player) effects(i int) {
	v := &p.channels[i]
	p.updateFunk(v)
	if v.effect == effectArpeggio && v.parameter == 0 {
		p.streams[i].Period = v.period
		return
	}
	if v.effect != effectExtended && v.effect >= effectTremolo {
		p.streams[i].Period = v.period
	}
	switch v.effect {
	case effectArpeggio:
		pitch := int(v.pitch)
		switch p.tick % 3 {
		case 1:
			pitch += int(v.parameter >> 4)
		case 2:
			pitch += int(v.parameter & 15)
		}
		if p.tick%3 == 0 {
			p.streams[i].Period = v.period
		} else if pitch <= 36 {
			p.streams[i].Period = p.module.Period(v.tune, pitch)
		}
	case effectPitchUp, effectPitchDown:
		p.slide(i, v.effect == effectPitchDown)
	case effectPortamento:
		if v.parameter != 0 {
			v.porta, v.parameter = byte(v.parameter), 0
		}
		p.portamento(i)
	case effectVibrato:
		v.vibrato = rememberNibbles(v.vibrato, v.parameter)
		p.vibrato(i)
	case effectPortamentoVolume:
		p.portamento(i)
		p.volumeSlide(i, false, false)
	case effectVibratoVolume:
		p.vibrato(i)
		p.volumeSlide(i, false, false)
	case effectTremolo:
		v.tremolo = rememberNibbles(v.tremolo, v.parameter)
		// The source's ramp sign uses the vibrato phase even for tremolo.
		delta := int(p.wave(v.tremoloPhase, v.waveforms>>4, v.vibratoPhase) * uint16(v.tremolo&15) >> 6)
		volume := int(v.volume)
		if int8(v.tremoloPhase) >= 0 {
			volume += delta
		} else {
			volume -= delta
		}
		p.streams[i].Volume = uint16(max(0, min(64, volume)))
		v.tremoloPhase += v.tremolo >> 2 & 60
	case effectVolumeSlide:
		p.volumeSlide(i, false, false)
	case effectExtended:
		p.extended(i)
	}
	// The original shared return overwrites the transient tremolo volume too.
	p.masterVolume(i)
}

func (p *Player) advanceFade() {
	if p.fadeStep == 0 {
		return
	}
	if p.master == p.fadeTarget {
		p.fadeStep = 0
		return
	}
	if p.master > p.fadeTarget {
		p.master -= p.fadeStep
		if p.master <= p.fadeTarget {
			p.master, p.fadeStep = p.fadeTarget, 0
		}
	} else {
		p.master += p.fadeStep
		if p.master >= p.fadeTarget {
			p.master, p.fadeStep = p.fadeTarget, 0
		}
	}
}

// update translates one original 50 Hz replay service, including its row/tick
// ordering and byte/word arithmetic. No processor or audio hardware is modeled.
func (p *Player) update() {
	if p.stopped {
		return
	}
	// The first active voice inherits the table's cadence. Later voices
	// inherit any volume remainder produced within this replay tick.
	p.repeatVolumeRemainder = -1
	p.tick++
	if p.tick < p.speed {
		for i := range p.channels {
			p.effects(i)
		}
		p.advanceFade()
		return
	}
	p.tick = 0
	if p.delay != 0 {
		for i := range p.channels {
			p.effects(i)
		}
	} else {
		for attempts := 0; attempts <= len(p.module.Orders); attempts++ {
			if int(p.order) < len(p.module.Orders) && p.module.Orders[p.order] < 128 {
				break
			}
			p.order, p.rowCursor, p.row = 0, 0, 0
		}
		pattern := p.module.Orders[p.order]
		if pattern < 0 || pattern >= len(p.module.Patterns) || p.rowCursor >= 64 {
			return
		}
		p.cursorPattern, p.previousPattern = pattern, pattern
		p.previousRowCursor = p.rowCursor
		notes := p.module.Patterns[pattern][p.rowCursor]
		for i, note := range notes {
			p.playNote(i, note)
		}
		p.rowCursor++

		for i := range p.channels {
			p.updateLoop(i)
		}
	}
	p.finishRow()
}

// finishRow applies the shared cursor changes after a complete source row.
func (p *Player) finishRow() {
	p.row++
	if p.pendingDelay != 0 {
		p.delay, p.pendingDelay = p.pendingDelay, 0
	}
	if p.delay != 0 {
		p.row--
		p.rowCursor = p.previousRowCursor
	}
	if p.patternLoop {
		p.patternLoop, p.breakRow = false, 0
	}
	if p.positionJump || p.nextOrder || p.row >= 64 {
		if !p.positionJump {
			p.order++
		}
		p.row, p.rowCursor, p.breakRow = 0, 0, 0
		p.positionJump, p.nextOrder = false, false
	}
	p.advanceFade()
}
