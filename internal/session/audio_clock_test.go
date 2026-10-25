package session

import (
	"bytes"
	"errors"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestFrameAudioConsumerFailureReachesItsProducingUpdate(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("PCM comparison failed")
	calls := 0
	s.EnableFrameAudio(func([]byte) error { calls++; return want })
	if err := s.Update(Input{}); !errors.Is(err, want) || calls != 1 {
		t.Fatalf("producer returned %v with %d sink calls", err, calls)
	}
}

func TestLateCueCannotReplaceEarlierWaitAudio(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			var sessions [2]*Session
			var output [2][]byte
			for index := range sessions {
				s, err := New(assets.Files())
				if err != nil {
					t.Fatal(err)
				}
				if err = s.Select(table); err != nil {
					t.Fatal(err)
				}
				s.EnableAppAudio(func(pcm []byte) error { output[index] = append(output[index][:0], pcm...); return nil })
				s.Sound.BeginPlay(true)
				sessions[index] = s
				// The input click precedes the common wait and replay service.
				s.Sound.Flipper()
				s.advanceAudio()
			}
			late := sessions[0].Table.Audio.SpringRelease.ReleaseCue(16)
			// This later game cue shares the same voice. Its request must not
			// rewrite audio belonging to the already elapsed common wait.
			sessions[0].Sound.Trigger(late)
			for _, s := range sessions {
				if err := s.finishAudioFrame(); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(output[0], output[1]) {
				t.Fatal("a late cue replaced the earlier input/wait audio")
			}
			if bytes.Equal(output[0], make([]byte, len(output[0]))) {
				t.Fatal("original flipper click did not sound during the wait")
			}
			// Applying that cue at the next block boundary must produce the
			// same following output: it is retained, rather than discarded.
			sessions[1].Sound.Trigger(late)
			for _, s := range sessions {
				s.advanceAudio()
				if err := s.finishAudioFrame(); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(output[0], output[1]) {
				t.Fatal("the following block lost or retimed the late cue")
			}
		})
	}
}

func TestAppAudioDurationDoesNotChangeNativeTableSteps(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			fixed, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			app, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			fixed.EnableFrameAudio(nil)
			blocks, samples := 0, 0
			app.EnableAppAudio(func(pcm []byte) error {
				if len(pcm) != 961*4 && len(pcm) != 962*4 {
					t.Fatalf("unexpected application table duration: %d bytes", len(pcm))
				}
				blocks++
				samples += len(pcm) / 4
				return nil
			})
			for _, s := range []*Session{fixed, app} {
				if err := s.Select(table); err != nil {
					t.Fatal(err)
				}
				s.Start(1)
			}
			for frame := range 800 {
				input := Input{Launch: frame < 400, Left: frame%19 < 7, Right: frame%23 < 9}
				for _, s := range []*Session{fixed, app} {
					if err := s.Update(input); err != nil {
						t.Fatal(err)
					}
				}
				if fixed.Rules.Frame != app.Rules.Frame || fixed.Engine.State != app.Engine.State || fixed.Camera != app.Camera || fixed.Charge != app.Charge || fixed.Players != app.Players || fixed.Sound.OutputDeclarations() != app.Sound.OutputDeclarations() {
					t.Fatalf("audio duration changed table or musical output declarations at step %d", frame)
				}
			}
			if blocks != 800 || int64(samples) != int64(blocks)*48000*71051/3546895 {
				t.Fatalf("application emitted %d blocks and %d stereo frames", blocks, samples)
			}
		})
	}
}
