package game

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

type bufferedAudioProbe struct {
	reader              io.Reader
	duration            time.Duration
	playing             bool
	plays, stops, reads int
	queued              []byte
	resets              int
}

func (p *bufferedAudioProbe) Play()                                { p.plays++; p.playing = true }
func (p *bufferedAudioProbe) PauseAndStopReading()                 { p.stops++; p.playing = false }
func (p *bufferedAudioProbe) SetBufferSize(duration time.Duration) { p.duration = duration }
func (p *bufferedAudioProbe) Reset()                               { p.resets++; p.playing = false; p.queued = nil }
func (p *bufferedAudioProbe) refill(t *testing.T) {
	t.Helper()
	if !p.playing {
		return
	}
	block := make([]byte, int(int64(p.duration)*sound.SampleRate/int64(time.Second))*4)
	n, err := p.reader.Read(block)
	if err != nil {
		t.Fatal(err)
	}
	p.queued = append(p.queued, block[:n]...)
	p.reads++
}

func TestBackendReadsPreserveProducedPCMWithoutAdvancingTheSession(t *testing.T) {
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	var playback audioPlayback
	playback.attach(s)
	want := make([]byte, playback.timing.reserveFrames*sound.FrameBytes)
	s.EnableAppAudio(func(pcm []byte) error {
		want = append(want, pcm...)
		playback.queue.Push(pcm)
		return nil
	})
	var got []byte
	requests := []int{1, 7, 511, 4096, 19200, 77}
	for frame := range 220 {
		if err := s.Update(session.Input{Launch: frame >= 120 && frame < 175, Left: frame%17 < 7, Right: frame%23 < 9}); err != nil {
			t.Fatal(err)
		}
		block := make([]byte, requests[frame%len(requests)])
		n, err := playback.queue.Read(block)
		if err != nil || n > sound.FrameBytes {
			t.Fatalf("queued read: %d, %v", n, err)
		}
		got = append(got, block[:n]...)
	}
	clock, pose, camera := s.Rules.Frame, s.Engine.State, s.Camera
	for {
		block := make([]byte, 19200)
		n, err := playback.queue.Read(block)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			break
		}
		got = append(got, block[:n]...)
	}
	if !bytes.Equal(got, want) || s.Rules.Frame != clock || s.Engine.State != pose || s.Camera != camera {
		t.Fatal("backend reads changed the produced sequence or advanced the game")
	}
	if _, err := s.Sound.Read(make([]byte, sound.FrameBytes)); !errors.Is(err, sound.ErrGameClock) {
		t.Fatal("a backend could advance the game-clock tracker directly")
	}
}

func TestOutputPauseKeepsQueuedPCMAndStopsProduction(t *testing.T) {
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	var playback audioPlayback
	playback.attach(s)
	probe := &bufferedAudioProbe{}
	open := func(reader io.Reader) (audioStream, error) { probe.reader = reader; return probe, nil }
	for range 8 {
		if err := s.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := playback.update(s.Sound, false, open); err != nil {
		t.Fatal(err)
	}
	probe.refill(t)
	if err := s.Update(session.Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	buffered := playback.queue.BufferedBytes()
	for range 64 {
		if err := s.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
		if err := playback.update(s.Sound, s.Paused, open); err != nil {
			t.Fatal(err)
		}
		probe.refill(t)
	}
	if playback.queue.BufferedBytes() != buffered || probe.reads != 1 || probe.stops != 1 {
		t.Fatal("paused output consumed or produced PCM")
	}
	if err := s.Update(session.Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	if err := playback.update(s.Sound, s.Paused, open); err != nil {
		t.Fatal(err)
	}
	if probe.plays != 2 || playback.queue.BufferedBytes() != buffered+sound.FrameBytes {
		t.Fatal("resuming did not produce exactly one new musical frame")
	}
}

func TestOriginalTablePauseRetiresQueuedPrePauseAudio(t *testing.T) {
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	for range 400 {
		if err = s.Update(session.Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	for count := 0; s.DefersKeyboardPause(); count++ {
		if count >= 256 {
			t.Fatal("normal launch did not enter live play")
		}
		if err = s.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	var playback audioPlayback
	playback.attach(s)
	probe := &bufferedAudioProbe{}
	open := func(reader io.Reader) (audioStream, error) { probe.reader = reader; return probe, nil }
	if err = playback.update(s.Sound, false, open); err != nil {
		t.Fatal(err)
	}
	// This obsolete marker must disappear when the original game stops voices.
	stale := bytes.Repeat([]byte{0x34, 0x12, 0x78, 0x56}, sound.FrameBytes/4)
	for range playback.timing.reserveFrames {
		probe.refill(t)
	}
	probe.queued = nil
	playback.queue.Push(stale)
	playback.queue.Push(stale)
	probe.refill(t)
	if !bytes.Contains(probe.queued, stale[:64]) {
		t.Fatal("backend did not prefetch the stale marker")
	}
	if err = s.Update(session.Input{PauseKeyPressed: true, SpringMotion: 100, SpringClick: true}); err != nil {
		t.Fatal(err)
	}
	if !s.Paused || s.AudioOutputPaused() {
		t.Fatal("did not enter the original running-output pause")
	}
	if err = playback.update(s.Sound, s.AudioOutputPaused(), open); err != nil {
		t.Fatal(err)
	}
	var queued []byte
	for {
		block := make([]byte, sound.FrameBytes)
		n, e := playback.queue.Read(block)
		if e != nil {
			t.Fatal(e)
		}
		if n == 0 {
			break
		}
		queued = append(queued, block[:n]...)
	}
	if bytes.Contains(queued, stale[:64]) {
		t.Fatal("pre-pause music remained queued after the original voice stop")
	}
	// The first original table interval spans 961 output stereo frames.
	if probe.resets != 1 || !probe.playing || len(queued) != playback.timing.reserveFrames*sound.FrameBytes+961*4 {
		t.Fatal("original pause did not reset once and retain its newly rendered wait")
	}
	if bytes.Equal(queued[len(queued)-sound.FrameBytes:], make([]byte, sound.FrameBytes)) {
		t.Fatal("reset discarded the new spring sound generated by the pause wait")
	}
	if len(probe.queued) != 0 {
		t.Fatal("backend-prefetched music survived the original pause")
	}
	if err = s.Update(session.Input{}); err != nil {
		t.Fatal(err)
	}
	if err = playback.update(s.Sound, s.AudioOutputPaused(), open); err != nil {
		t.Fatal(err)
	}
	if probe.resets != 1 {
		t.Fatal("each paused wait discarded fresh pause audio")
	}
}

func TestModuleReplacementRetainsOneOutputAndQueuedOldPCM(t *testing.T) {
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	var playback audioPlayback
	playback.attach(s)
	probe, opens := &bufferedAudioProbe{}, 0
	open := func(reader io.Reader) (audioStream, error) { opens++; probe.reader = reader; return probe, nil }
	if err := s.Update(session.Input{}); err != nil {
		t.Fatal(err)
	}
	first := s.Sound
	if err := playback.update(first, false, open); err != nil {
		t.Fatal(err)
	}
	buffered := playback.queue.BufferedBytes()
	if err := s.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	if err := playback.update(s.Sound, true, open); err != nil {
		t.Fatal(err)
	}
	if first == s.Sound || opens != 1 || probe.stops != 1 || playback.queue.BufferedBytes() != buffered {
		t.Fatal("module change reopened output or discarded its previous PCM")
	}
	if err := s.Update(session.Input{}); err != nil {
		t.Fatal(err)
	}
	if err := playback.update(s.Sound, false, open); err != nil {
		t.Fatal(err)
	}
	if opens != 1 || probe.plays != 2 || playback.queue.BufferedBytes() != buffered+sound.FrameBytes {
		t.Fatal("the next module failed to append to the continuous output")
	}
}

func TestOutputCreationErrorPreservesProducedPCM(t *testing.T) {
	playback := audioPlayback{queue: sound.NewPCMQueue(0)}
	playback.queue.Push([]byte{1, 2, 3, 4})
	want := errors.New("output unavailable")
	err := playback.update(&sound.Player{}, false, func(io.Reader) (audioStream, error) { return nil, want })
	if !errors.Is(err, want) || playback.stream != nil || playback.queue.BufferedBytes() != 4 {
		t.Fatal("output creation error hid its failure or discarded produced PCM")
	}
}

func TestSuspendStopsTheAttachedAudioOutput(t *testing.T) {
	source := &sound.Player{}
	probe := &bufferedAudioProbe{playing: true}
	g := &Game{Session: &session.Session{Sound: source}, audio: audioPlayback{source: source, stream: probe}}
	g.Suspend()
	if !g.Session.Paused || probe.playing || probe.stops != 1 {
		t.Fatal("suspending the host left its session or audio output running")
	}
}

func TestFocusLossFreezesProductionAndRetainsQueuedPCM(t *testing.T) {
	g, err := New(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	g.audio.attach(g.Session)
	probe := &bufferedAudioProbe{playing: true}
	g.audio.stream = probe
	g.updateFocus(true)
	for range 12 {
		if err := g.Session.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	state, buffered := g.Session.Intro.State(), g.audio.queue.BufferedBytes()
	for range 100 {
		if g.updateFocus(false) {
			t.Fatal("unfocused host would advance its session")
		}
	}
	if !g.Session.Paused || g.audio.queue.BufferedBytes() != buffered || g.Session.Intro.State().Remaining != state.Remaining || probe.stops != 1 {
		t.Fatal("focus loss changed the clock, queued audio or repeated suspension")
	}
	if !g.updateFocus(true) || !g.Session.Paused || !g.controls.waitForRelease {
		t.Fatal("regaining focus bypassed the explicit resume/release boundary")
	}
}
