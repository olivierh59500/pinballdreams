package sound

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/olivierh59500/pinballdreams/internal/audiodata"
)

const (
	SampleRate = 48000
	TickRate   = 50
	FrameBytes = SampleRate / TickRate * 4
)

var ErrGameClock = errors.New("game-clock audio must be read from its PCM queue")

type renderedVoice struct {
	start, length, index int
	generation           uint32
	remaining            int64
	value                int8
	period               uint16
	holding              bool
}

// Player owns the native tracker and signed stereo PCM renderer. The game
// advances replay explicitly; standalone exports may use the streaming reader.
type Player struct {
	mu                                                        sync.Mutex
	module                                                    *Module
	data                                                      []byte
	sampleFingerprint                                         uint32
	channels                                                  [4]channel
	streams                                                   [4]streamVoice
	rendered                                                  [4]renderedVoice
	order, savedOrder, row, rowCursor, previousRowCursor      uint16
	cursorPattern, previousPattern                            int
	speed, tick, breakRow, fineMask                           byte
	priority, repeats, master, fadeTarget, fadeStep           byte
	presentedMaster                                           byte
	pendingDelay, delay                                       byte
	positionJump, nextOrder, patternLoop, stopped, returnStop bool
	filter                                                    bool
	samplesToTick                                             int
	muted, paused                                             bool
	outputFilters                                             [2]outputFilter
	pendingPCM                                                [4]byte
	pendingOffset, pendingLength                              int
	commandObserver                                           func(Command)
	retriggerPhase                                            [16]byte
	repeatVolumeRemainder                                     int
	gameClock                                                 bool
}

func New(module *Module, order int) *Player {
	p := &Player{module: module, data: append([]byte(nil), module.PCM...), master: 64, presentedMaster: 64, filter: true}
	for i := range p.channels {
		p.channels[i].sampleStart, p.channels[i].loopStart, p.channels[i].funkPosition = -1, -1, -1
		p.streams[i].Start, p.streams[i].LoopStart = -1, -1
	}
	for i, value := range p.data {
		p.sampleFingerprint += uint32(i+1) * uint32(value)
	}

	if order < 0 {
		order = module.InitialOrder
	}
	p.selectMusic(order, 0, 0)
	return p
}

func (p *Player) Mute(muted bool)   { p.mu.Lock(); p.muted = muted; p.mu.Unlock() }
func (p *Player) Pause(paused bool) { p.mu.Lock(); p.paused = paused; p.mu.Unlock() }

// StopVoices silences current output segments without resetting tracker rows,
// effects or return-song state. Later note events can start those voices again.
func (p *Player) StopVoices() {
	p.mu.Lock()
	for i := range p.streams {
		p.streams[i].Enabled, p.streams[i].Volume = false, 0
	}
	p.mu.Unlock()
}

// LowPassEnabled reports the current optional tone filter. Tracker effects
// change this setting, which survives a change of music in the original.
func (p *Player) LowPassEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.filter
}

// SetLowPass restores an inherited tone selection without restarting music.
func (p *Player) SetLowPass(enabled bool) {
	p.mu.Lock()
	p.filter = enabled
	p.mu.Unlock()
}

// SetRetriggerPhase follows the original table-bank/repeated-note coupling.
// The extractor supplies musical remainders, not source pointers or registers.
func (p *Player) SetRetriggerPhase(phase [16]byte) {
	p.mu.Lock()
	p.retriggerPhase = phase
	p.mu.Unlock()
}

// BeginBall selects the normal/quiet source song followed by its returning jingle.
func (p *Player) BeginBall(musicDisabled bool) {
	for index, command := range p.module.BallStart {
		if index == 0 && musicDisabled {
			command = p.module.QuietMusic
		}
		p.Trigger(command)
	}
}

// BeginPlay follows the original normal/quiet song selection at the launch gate.
// The quiet song preserves event samples and their original channel behavior.
func (p *Player) BeginPlay(musicDisabled bool) {
	command := p.module.PlayMusic
	if musicDisabled {
		command = p.module.QuietMusic
	}
	p.Trigger(command)
}

// Flipper plays the source key-edge click on the original event voice.
func (p *Player) Flipper() {
	p.Trigger(p.module.Flipper)
}

// ReleaseSpring uses the supplied sample and the source power-level volume.
// A release below the first visible power step makes no sound declaration.
func (p *Player) ReleaseSpring(level int) {
	command := p.module.SpringRelease.ReleaseCue(level)
	if command.Action != audiodata.None {
		p.Trigger(command)
	}
}

// ObserveCommands installs an optional trace hook for development comparisons.
// The hook runs after applying each command, outside the playback mutex.
func (p *Player) ObserveCommands(observer func(Command)) {
	p.mu.Lock()
	p.commandObserver = observer
	p.mu.Unlock()
}

// Fade follows the original byte-valued master-volume target and increment.
func (p *Player) Fade(target, step byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fadeTarget, p.fadeStep = target, step
}

// Trigger applies a decoded game cue. Music selection and event notes share
// the native channel state, retaining the supplied priority/return behavior.
func (p *Player) Trigger(command Command) {
	p.mu.Lock()
	switch command.Action {
	case audiodata.Sound:
		p.playNote(3, command.Note)
		p.updateLoop(3)
	case audiodata.Music:
		p.selectMusic(command.Order, command.Repeats, byte(command.Priority))
	}
	observer := p.commandObserver
	p.mu.Unlock()
	if observer != nil && command.Action != audiodata.None && command.Action != "" {
		observer(command)
	}
}

func (p *Player) renderFrame(out []byte) {
	if !p.paused {
		if p.samplesToTick <= 0 {
			p.update()
			p.samplesToTick = SampleRate / TickRate
		}
		p.samplesToTick--
	}
	p.mixFrame(out)
}

func (p *Player) mixFrame(out []byte) {
	clear(out)
	if p.paused {
		return
	}
	var left, right float64
	for i := range p.streams {
		amplitude := p.mixVoice(i)
		if i == 0 || i == 3 {
			left += amplitude
		} else {
			right += amplitude
		}
	}
	left, right = p.filterSample(left, right)
	if p.muted {
		return
	}
	binary.LittleEndian.PutUint16(out, uint16(int16(max(-32768, min(32767, left)))))
	binary.LittleEndian.PutUint16(out[2:], uint16(int16(max(-32768, min(32767, right)))))
}

// Read preserves partial stereo frames so all read sizes produce the same PCM.
func (p *Player) Read(out []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.gameClock {
		return 0, ErrGameClock
	}
	for offset := 0; offset < len(out); {
		if p.pendingLength == 0 {
			p.renderFrame(p.pendingPCM[:])
			p.pendingOffset, p.pendingLength = 0, 4
		}
		n := min(p.pendingLength, len(out)-offset)
		copy(out[offset:offset+n], p.pendingPCM[p.pendingOffset:p.pendingOffset+n])
		p.pendingOffset += n
		p.pendingLength -= n
		offset += n
	}
	return len(out), nil
}

// OutputState carries the continuous stereo filter history across modules.
// A single game-thread renderer owns it; device readers receive only PCM.
type OutputState struct{ filters [2]outputFilter }

// AdvanceFrame runs one replay service at the game's declared input phase.
// Switching to this mode prevents a device read from advancing replay again.
func (p *Player) AdvanceFrame() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gameClock = true
	if !p.paused {
		p.update()
	}
}

// HoldFrame selects the game clock without advancing tracker rows or effects.
// Output still renders, including filter tails and newly requested event notes.
func (p *Player) HoldFrame() {
	p.mu.Lock()
	p.gameClock = true
	p.mu.Unlock()
}

// RenderFrame mixes one 20 ms block after this frame's game cues. It never
// advances replay. The caller retains output history when selecting new music.
func (p *Player) RenderFrame(out []byte, state *OutputState) {
	if len(out) != FrameBytes || state == nil {
		panic("sound: RenderFrame requires one PCM frame and continuous output state")
	}
	p.RenderFrames(out, state)
}

// RenderFrames mixes complete stereo frames without moving the replay clock.
// A fractional game cadence can vary block length while sample pitch and
// continuous segment/filter history retain the fixed output sample rate.
func (p *Player) RenderFrames(out []byte, state *OutputState) {
	if len(out)%4 != 0 || state == nil {
		panic("sound: RenderFrames requires complete stereo frames and output state")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.gameClock {
		panic("sound: AdvanceFrame must select the game clock before rendering")
	}
	p.outputFilters = state.filters
	for at := 0; at < len(out); at += 4 {
		p.mixFrame(out[at : at+4])
	}
	state.filters = p.outputFilters
}
