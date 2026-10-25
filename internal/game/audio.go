package game

import (
	"io"
	"runtime"
	"time"

	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

type outputTiming struct {
	device, player time.Duration
	reserveFrames  int
}

func timingForPlatform(platform string) outputTiming {
	if platform == "darwin" {
		// Four 5 ms device blocks prime within one game tick. Two queued
		// ticks cover the callback and the measured scheduling margin.
		return outputTiming{device: 20 * time.Millisecond, player: 40 * time.Millisecond, reserveFrames: 2}
	}
	// Android chooses its actual burst size. Seven reserve blocks cover two
	// 60 ms backend reads plus the tested 16 ms game-thread scheduling jitter.
	return outputTiming{device: 10 * time.Millisecond, player: 100 * time.Millisecond, reserveFrames: 7}
}

var platformOutputTiming = timingForPlatform(runtime.GOOS)

type audioStream interface {
	Play()
	PauseAndStopReading()
	SetBufferSize(time.Duration)
	Reset()
}

// audioPlayback consumes one continuous PCM queue across module changes.
// Replay and mixing belong to session updates, never to a backend refill.
type audioPlayback struct {
	source      *sound.Player
	stream      audioStream
	queue       *sound.PCMQueue
	paused      bool
	attached    bool
	timing      outputTiming
	manualPause bool
	diagnostics audioDiagnostics
}

func (p *audioPlayback) attach(s *session.Session) {
	if p.attached {
		return
	}
	p.timing = platformOutputTiming
	p.queue = sound.NewPCMQueue(p.timing.reserveFrames)
	p.queue.SetMetricsEnabled(p.diagnostics.enabled)
	s.EnableAppAudio(func(samples []byte) error {
		manualPause := s.Paused && !s.AudioOutputPaused()
		if manualPause && !p.manualPause {
			// The original pause stops voices immediately. Retire older PCM
			// before appending this wait's filter tail or new spring/click cue.
			if p.stream != nil {
				p.stream.Reset()
				p.paused = true
			}
			p.queue.Reset(p.timing.reserveFrames)
		}
		p.manualPause = manualPause
		p.queue.Push(samples)
		return nil
	})
	p.attached = true
}

func (p *audioPlayback) update(source *sound.Player, paused bool, open func(io.Reader) (audioStream, error)) error {
	p.source = source
	if source == nil {
		p.setPaused(true)
		return nil
	}
	if p.stream == nil {
		if p.timing.player == 0 {
			p.timing = platformOutputTiming
		}
		if p.queue == nil {
			p.queue = sound.NewPCMQueue(p.timing.reserveFrames)
			p.queue.SetMetricsEnabled(p.diagnostics.enabled)
		}
		stream, err := open(p.queue)
		if err != nil {
			return err
		}
		stream.SetBufferSize(p.timing.player)
		p.stream, p.paused = stream, paused
		if paused {
			stream.PauseAndStopReading()
		} else {
			stream.Play()
		}
	}
	p.setPaused(paused)
	return nil
}

func (p *audioPlayback) setPaused(paused bool) {
	if p.stream == nil || p.paused == paused {
		return
	}
	if paused {
		p.stream.PauseAndStopReading()
	} else {
		p.stream.Play()
	}
	p.paused = paused
}

func (p *audioPlayback) discardQueued() {
	if p.stream != nil {
		p.stream.Reset()
		p.paused = true
	}
	if p.queue != nil {
		p.queue.Reset(p.timing.reserveFrames)
	}
}
