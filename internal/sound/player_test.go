package sound

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func sourceModule(t *testing.T, id int) *Module {
	t.Helper()
	table, err := resources.LoadFS(assets.Files(), id)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Decode(table)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSourceEntryAndBallMusic(t *testing.T) {
	initial := []int{4, 27, 10, 10}
	jingles := []int{31, 61, 27, 47}
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			m := sourceModule(t, id)
			if m.InitialMusicDisabled {
				t.Fatal("source table entered with music disabled")
			}
			if m.InitialOrder != initial[id] {
				t.Fatalf("initial song %d; want %d", m.InitialOrder, initial[id])
			}
			p := New(m, -1)
			p.BeginBall(false)
			if int(p.order) != jingles[id] || p.priority != 200 || p.repeats != 1 || p.savedOrder != 0 {
				t.Fatalf("normal ball music order=%d priority=%d repeats=%d return=%d", p.order, p.priority, p.repeats, p.savedOrder)
			}
			p.BeginPlay(false)
			if p.order != uint16(m.PlayMusic.Order) || p.priority != 0 {
				t.Fatal("launch gate did not restore the principal game music")
			}
			p.BeginPlay(true)
			if p.order != uint16(m.QuietMusic.Order) || p.muted {
				t.Fatal("quiet song selection muted event audio")
			}
			p.Read(make([]byte, 400))
			before := p.streams[3].Starts
			p.Flipper()
			if p.streams[3].Starts != before+1 || !p.streams[3].Enabled {
				t.Fatal("original flipper click did not start the event voice")
			}
		})
	}
}

func TestPCMIndependentOfReadBoundaries(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			m := sourceModule(t, id)
			want := make([]byte, SampleRate*4)
			if _, err := io.ReadFull(New(m, -1), want); err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(want, make([]byte, len(want))) {
				t.Fatal("source song rendered only silence")
			}
			p := New(m, -1)
			got := make([]byte, len(want))
			sizes := []int{1, 7, 4, 511, 2048, 3}
			for offset, part := 0, 0; offset < len(got); part++ {
				n := min(sizes[part%len(sizes)], len(got)-offset)
				if _, err := io.ReadFull(p, got[offset:offset+n]); err != nil {
					t.Fatal(err)
				}
				offset += n
			}
			if !bytes.Equal(got, want) {
				t.Fatal("splitting PCM reads changed playback")
			}
		})
	}
}

func TestOutputDeclarationObservationPreservesPlayback(t *testing.T) {
	m := sourceModule(t, 0)
	p, reference := New(m, -1), New(m, -1)
	got, want := make([]byte, FrameBytes), make([]byte, FrameBytes)
	for frame := range 12 {
		if frame == 4 {
			p.Flipper()
			reference.Flipper()
		}
		observed := p.OutputDeclarations()
		observed[3].Enabled = false
		observed[3].Volume = 0
		p.Read(got)
		reference.Read(want)
		if !bytes.Equal(got, want) {
			t.Fatalf("observing output changed PCM at frame %d", frame)
		}
	}
}

func TestPauseAndMuteClockBehavior(t *testing.T) {
	m := sourceModule(t, 0)
	p, reference := New(m, -1), New(m, -1)
	prefix := make([]byte, 7000)
	p.Read(prefix)
	reference.Read(prefix)
	p.Pause(true)
	paused := make([]byte, 12000)
	p.Read(paused)
	if !bytes.Equal(paused, make([]byte, len(paused))) {
		t.Fatal("paused playback emitted audio")
	}
	p.Pause(false)
	got, want := make([]byte, 12000), make([]byte, 12000)
	p.Read(got)
	reference.Read(want)
	if !bytes.Equal(got, want) {
		t.Fatal("pause advanced the replay clock")
	}
	p.Mute(true)
	p.Read(got)
	reference.Read(want)
	if !bytes.Equal(got, make([]byte, len(got))) {
		t.Fatal("mute emitted audio")
	}
	p.Mute(false)
	p.Read(got)
	reference.Read(want)
	if !bytes.Equal(got, want) {
		t.Fatal("muted playback did not preserve the continuing replay clock")
	}
}

func TestFunkUsesPrivateSampleStorage(t *testing.T) {
	m := sourceModule(t, 0)
	original := append([]byte(nil), m.PCM...)
	p := New(m, -1)
	before := append([]byte(nil), p.data...)
	// This is an input command for the original EFx handler, not new PCM data.
	p.Trigger(Command{Action: audiodata.Sound, Note: Note{Sample: 1, Pitch: 13, Effect: 14, Parameter: 255}})
	if bytes.Equal(before, p.data) {
		t.Fatal("invert-loop command did not change private sample data")
	}
	if !bytes.Equal(original, m.PCM) {
		t.Fatal("replay mutated the immutable reconstructed module")
	}
}

func TestOriginalStereoRoutingHasNoAddedCrossfeed(t *testing.T) {
	m := sourceModule(t, 0)
	for index := range 4 {
		p := New(m, -1)
		p.stopped, p.filter = true, false
		sample := m.Samples[0]
		p.streams[index] = streamVoice{Start: sample.Offset, Length: sample.Length, LoopStart: sample.Offset, LoopLength: 2, Period: 428, Volume: 64, Starts: 1, Enabled: true}
		pcm := make([]byte, SampleRate/4*4)
		p.Read(pcm)
		var energy [2]uint64
		for at := 0; at < len(pcm); at += 4 {
			for side := range 2 {
				value := int64(int16(binary.LittleEndian.Uint16(pcm[at+side*2:])))
				energy[side] += uint64(value * value)
			}
		}
		wanted := 1
		if index == 0 || index == 3 {
			wanted = 0
		}
		if energy[wanted] == 0 || energy[1-wanted] != 0 {
			t.Fatalf("channel %d stereo energy=%v", index, energy)
		}
	}
}

func TestPlaybackDoesNotUseSourceProvenance(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			m := sourceModule(t, id)
			plain := *m
			plain.Provenance = audiodata.Provenance{}
			want, got := make([]byte, SampleRate*4), make([]byte, SampleRate*4)
			New(m, -1).Read(want)
			New(&plain, -1).Read(got)
			if !bytes.Equal(got, want) {
				t.Fatal("playback depends on original addresses or packed-row positions")
			}
		})
	}
}
