package game

import (
	"testing"
	"time"

	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

type diagnosticBufferedProbe struct {
	bufferedAudioProbe
	buffered, calls int
}

func (p *diagnosticBufferedProbe) BufferedSize() int { p.calls++; return p.buffered }

func TestAudioDiagnosticsStayDisabledUntilExplicitlyEnabled(t *testing.T) {
	queue := sound.NewPCMQueue(0)
	probe := &diagnosticBufferedProbe{buffered: 480}
	g := &Game{audio: audioPlayback{queue: queue, stream: probe}}
	if _, ready := g.PollAudioDiagnostics(time.Unix(100, 0)); ready || probe.calls != 0 || queue.Metrics().Enabled {
		t.Fatal("ordinary playback enabled or sampled diagnostic accounting")
	}
	g.SetAudioDiagnostics(true)
	if !queue.Metrics().Enabled {
		t.Fatal("explicit diagnostics did not enable queue accounting")
	}
	g.SetAudioDiagnostics(false)
	if _, ready := g.PollAudioDiagnostics(time.Unix(101, 0)); ready || probe.calls != 0 || queue.Metrics().Enabled {
		t.Fatal("disabled diagnostics continued polling the software reader")
	}
}

func TestAudioDiagnosticsCoverAttachmentAndOptionalPlayer(t *testing.T) {
	g := &Game{Session: &session.Session{}}
	g.SetAudioDiagnostics(true)
	if _, ready := g.PollAudioDiagnostics(time.Unix(100, 0)); ready || g.audio.queue != nil {
		t.Fatal("diagnostic polling created an output queue")
	}
	g.audio.attach(g.Session)
	if metrics := g.audio.queue.Metrics(); !metrics.Enabled || metrics.InitialBufferedBytes != uint64(g.audio.timing.reserveFrames*sound.FrameBytes) {
		t.Fatal("enabling before attachment missed the initial silent reserve")
	}
	snapshot, ready := g.PollAudioDiagnostics(time.Unix(100, 0))
	if !ready || snapshot.PlayerBufferedBytes != nil || snapshot.PlayerBufferedMilliseconds != nil || snapshot.ObservedSoftwareBufferedBytes != snapshot.Queue.BufferedBytes {
		t.Fatal("an unavailable player buffer was reported as measured")
	}
}

func TestAudioDiagnosticsReportSoftwareDequeueAndPollOnlyOncePerSecond(t *testing.T) {
	queue := sound.NewPCMQueue(0)
	probe := &diagnosticBufferedProbe{buffered: 480}
	g := &Game{audio: audioPlayback{queue: queue, stream: probe}}
	g.SetAudioDiagnostics(true)
	queue.Push(make([]byte, 1920))
	if n, err := queue.Read(make([]byte, 960)); err != nil || n != 960 {
		t.Fatal("software reader did not remove its requested samples")
	}
	start := time.Unix(100, 0)
	snapshot, ready := g.PollAudioDiagnostics(start)
	if !ready || snapshot.Queue.SourceReadBytes != 960 || snapshot.Queue.BufferedBytes != 960 || snapshot.Queue.BufferedDuration != 5*time.Millisecond {
		t.Fatalf("software queue snapshot differs: %+v", snapshot)
	}
	if snapshot.PlayerBufferedBytes == nil || *snapshot.PlayerBufferedBytes != 480 || *snapshot.PlayerBufferedMilliseconds != 2.5 || snapshot.QueueBufferedMilliseconds != 5 || snapshot.ObservedSoftwareBufferedBytes != 1440 || snapshot.ObservedSoftwareBufferedMilliseconds != 7.5 {
		t.Fatalf("software buffer sizes/durations differ: %+v", snapshot)
	}
	if snapshot.Scope != "software-source-queue-and-player-only" || !snapshot.BuffersSampledSeparately {
		t.Fatal("snapshot lost its software-only and non-atomic bounds")
	}
	if _, ready := g.PollAudioDiagnostics(start.Add(999 * time.Millisecond)); ready || probe.calls != 1 {
		t.Fatal("diagnostics sampled more than once per second")
	}
	if _, ready := g.PollAudioDiagnostics(start.Add(time.Second)); !ready || probe.calls != 2 {
		t.Fatal("next diagnostic interval was not sampled")
	}
	if queue.BufferedBytes() != 960 {
		t.Fatal("inspecting software buffers consumed queued PCM")
	}
	g.SetAudioDiagnostics(false)
	g.SetAudioDiagnostics(true)
	snapshot, ready = g.PollAudioDiagnostics(start.Add(time.Second))
	if !ready || snapshot.Queue.InitialBufferedBytes != 960 || snapshot.Queue.SourceReadBytes != 0 {
		t.Fatal("reenabling did not start a new software-accounting epoch")
	}
}
