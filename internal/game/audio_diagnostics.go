package game

import (
	"time"

	"github.com/olivierh59500/pinballdreams/internal/sound"
)

// AudioDiagnosticsSnapshot describes software buffering only. Queue reads can
// prefetch into the player; they are not device consumption or underruns. The
// separately sampled buffers exclude the driver/FIFO and audible latency.
type AudioDiagnosticsSnapshot struct {
	Scope                                string
	Queue                                sound.PCMQueueMetrics
	QueueBufferedMilliseconds            float64
	PlayerBufferedBytes                  *int     `json:",omitempty"`
	PlayerBufferedMilliseconds           *float64 `json:",omitempty"`
	ObservedSoftwareBufferedBytes        int
	ObservedSoftwareBufferedMilliseconds float64
	BuffersSampledSeparately             bool
	OutputPaused                         bool
	DeviceInitializationPending          bool
	InitializationWaitHostPolls          uint64
}

type audioDiagnostics struct {
	enabled        bool
	nextPoll       time.Time
	readyWaitPolls uint64
}

// SetAudioDiagnostics is called on the game thread. Ordinary/release playback
// leaves metrics disabled; enabling before attachment also covers priming reads.
func (g *Game) SetAudioDiagnostics(enabled bool) {
	d := &g.audio.diagnostics
	if d.enabled == enabled {
		return
	}
	d.enabled, d.nextPoll = enabled, time.Time{}
	d.readyWaitPolls = 0
	if g.audio.queue != nil {
		g.audio.queue.SetMetricsEnabled(enabled)
	}
}

// PollAudioDiagnostics samples at most once per second on the game thread.
// The caller supplies its clock; the software reader performs no timed work.
func (g *Game) PollAudioDiagnostics(now time.Time) (AudioDiagnosticsSnapshot, bool) {
	p := &g.audio
	if !p.diagnostics.enabled || p.queue == nil || !p.diagnostics.nextPoll.IsZero() && now.Before(p.diagnostics.nextPoll) {
		return AudioDiagnosticsSnapshot{}, false
	}
	p.diagnostics.nextPoll = now.Add(time.Second)
	metrics := p.queue.Metrics()
	result := AudioDiagnosticsSnapshot{
		Scope: "software-source-queue-and-player-only", Queue: metrics,
		QueueBufferedMilliseconds:     float64(metrics.BufferedDuration) / float64(time.Millisecond),
		ObservedSoftwareBufferedBytes: metrics.BufferedBytes,
		BuffersSampledSeparately:      true, OutputPaused: p.paused,
		DeviceInitializationPending: g.context != nil && !g.context.initialized(),
		InitializationWaitHostPolls: p.diagnostics.readyWaitPolls,
	}
	if stream, ok := p.stream.(interface{ BufferedSize() int }); ok {
		buffered := max(0, stream.BufferedSize())
		milliseconds := float64(buffered) * 1000 / (sound.SampleRate * 4)
		result.PlayerBufferedBytes, result.PlayerBufferedMilliseconds = &buffered, &milliseconds
		result.ObservedSoftwareBufferedBytes += buffered
	}
	result.ObservedSoftwareBufferedMilliseconds = float64(result.ObservedSoftwareBufferedBytes) * 1000 / (sound.SampleRate * 4)
	return result, true
}
