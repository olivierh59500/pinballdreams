package game

import (
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

// audioDevice exposes the device-buffer request absent from Ebitengine's
// audio context. It uses the same Oto driver and receives native stereo PCM.
type audioDevice struct {
	context *oto.Context
	ready   <-chan struct{}
	mu      sync.Mutex
	err     error
}

type deviceAudioStream struct{ *oto.Player }

// Reset quiesces source reads and discards prefetched samples. Oto's retained
// Reset operation is needed here because the continuous PCM queue cannot seek.
// Already submitted device/FIFO samples retain their backend latency.
func (s deviceAudioStream) Reset() { s.Player.Reset() }

type audioFocusContext interface {
	Suspend() error
	Resume() error
}

type audioFocusState struct{ active bool }

func (s *audioFocusState) update(active bool, context audioFocusContext) error {
	if active == s.active {
		return nil
	}
	var err error
	if active {
		err = context.Resume()
	} else {
		err = context.Suspend()
	}
	if err == nil {
		s.active = active
	}
	return err
}

func (s deviceAudioStream) SetBufferSize(duration time.Duration) {
	s.Player.SetBufferSize(int(int64(duration) * sound.SampleRate * 4 / int64(time.Second)))
}

func newAudioDevice(timing outputTiming) (*audioDevice, error) {
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: sound.SampleRate, ChannelCount: 2, Format: oto.FormatSignedInt16LE,
		BufferSize: timing.device, ApplicationName: "Pinball Dreams",
	})
	if err != nil {
		return nil, err
	}
	d := &audioDevice{context: context, ready: ready}
	go d.followFocus(ebiten.IsFocused)
	return d, nil
}

func (d *audioDevice) error() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		return d.err
	}
	return d.context.Err()
}

func (d *audioDevice) initialized() bool {
	select {
	case <-d.ready:
		return true
	default:
		return false
	}
}

func (d *audioDevice) open(source io.Reader) (audioStream, error) {
	if err := d.error(); err != nil {
		return nil, err
	}
	return deviceAudioStream{d.context.NewPlayer(source)}, nil
}

// followFocus also runs while mobile updates are stopped. IsFocused is the
// public, concurrent-safe foreground signal on both desktop and mobile.
// All device calls stay off the Android UI thread.
func (d *audioDevice) followFocus(focused func() bool) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	ready := false
	state := audioFocusState{active: true}
	for range ticker.C {
		if d.context.Err() != nil {
			return
		}
		if !ready {
			select {
			case <-d.ready:
				ready = true
			default:
				continue
			}
		}
		err := state.update(focused(), d.context)
		if err != nil {
			d.mu.Lock()
			d.err = err
			d.mu.Unlock()
			return
		}
	}
}
