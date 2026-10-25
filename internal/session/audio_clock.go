package session

import (
	"github.com/olivierh59500/pinballdreams/internal/sound"
	"github.com/olivierh59500/pinballdreams/internal/timing"
)

// frameAudio owns replay, wait audio and continuous output history. A backend
// consumes completed PCM without moving the game or its musical state.
type frameAudio struct {
	pending  *sound.Player
	output   sound.OutputState
	pcm      []byte
	sink     func([]byte) error
	waitOnly bool
	appClock bool
	samples  timing.Samples
	bytes    int
	rendered bool
}

// EnableFrameAudio makes this session own the 50 Hz replay clock. The sink
// receives a reusable 20 ms block and must copy it before returning.
// A sink error is returned by the update that produced that block.
func (s *Session) EnableFrameAudio(sink func([]byte) error) {
	s.frameAudio = &frameAudio{sink: sink, pcm: make([]byte, sound.FrameBytes)}
}

// EnableAppAudio gives table steps their supplied fractional duration. Opening
// and menu scenes retain their nominal cadence. Fixed 50 Hz oracle/export
// comparisons use EnableFrameAudio explicitly, outside the application host.
func (s *Session) EnableAppAudio(sink func([]byte) error) {
	s.frameAudio = &frameAudio{sink: sink, appClock: true,
		pcm: make([]byte, timing.Table.MaxSamples(sound.SampleRate)*4)}
}

func (s *Session) beginAudioBlock() {
	a := s.frameAudio
	a.rendered = false
	a.bytes = sound.FrameBytes
	if a.appClock {
		cadence := timing.Nominal
		if s.Table != nil {
			cadence = timing.Table
		}
		a.bytes = a.samples.Next(cadence, sound.SampleRate) * 4
	}
}

func (s *Session) advanceAudio() {
	if a := s.frameAudio; a != nil && a.pending == nil && s.Sound != nil && !s.Paused {
		a.pending = s.Sound
		s.beginAudioBlock()
		if a.appClock && s.Table != nil {
			// Preserve the input/wait audio before later game cues can replace
			// its voice. Replay separates the two logical wait portions. The
			// midpoint is a native schedule, not a source CPU timing model.
			midpoint := a.bytes / 8 * 4
			a.pending.HoldFrame()
			a.pending.RenderFrames(a.pcm[:midpoint], &a.output)
			a.pending.AdvanceFrame()
			a.pending.RenderFrames(a.pcm[midpoint:a.bytes], &a.output)
			a.rendered = true
			return
		}
		a.pending.AdvanceFrame()
	}
}

func (s *Session) finishAudioFrame() error {
	if s.frameAudio.waitOnly {
		s.frameAudio.waitOnly = false
		s.frameAudio.pending = s.Sound
		s.beginAudioBlock()
		s.Sound.HoldFrame()
	} else {
		// Host-only transitions can return without a table wait. Their unpaused
		// application tick still owns one musical frame, including module changes.
		s.advanceAudio()
	}
	a := s.frameAudio
	if a.pending == nil {
		return nil
	}
	if !a.rendered {
		a.pending.RenderFrames(a.pcm[:a.bytes], &a.output)
	}
	a.pending = nil
	if a.sink != nil {
		return a.sink(a.pcm[:a.bytes])
	}
	return nil
}

// AudioOutputPaused keeps the output running through a manual table pause so
// stopped-note tails decay. System suspension still stops reading queued PCM.
func (s *Session) AudioOutputPaused() bool {
	return s.Paused && (s.tablePause == nil || s.tablePause.suspended)
}
