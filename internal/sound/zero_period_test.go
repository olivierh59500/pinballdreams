package sound

import (
	"math"
	"testing"
)

func TestZeroPeriodKeepsPendingValueAndResumesAtNextSourceClock(t *testing.T) {
	p := &Player{data: []byte{10, 70, 110}}
	p.streams[0] = streamVoice{Start: 0, Length: 3, LoopStart: 0, LoopLength: 3, Period: 428, Volume: 64, Starts: 1, Enabled: true}
	p.mixVoice(0)
	p.streams[0].Period = 0
	for frame := 1; frame <= 16; frame++ {
		got := p.mixVoice(0)
		if frame < 5 && got != 10*128 {
			t.Fatal("zero period replaced the already scheduled value")
		}
		if frame >= 6 && got != 70*128 {
			t.Fatalf("zero-period frame%d: %g; want held value %d", frame, got, 70*128)
		}
	}
	p.data[1] = 120
	if got := p.mixVoice(0); got != 70*128 {
		t.Fatal("a sample-buffer mutation changed the held output value")
	}
	p.streams[0].Period = 213
	want := float64(70*SampleRate+110*(pitchClock-SampleRate)) * 128 / float64(pitchClock)
	if got := p.mixVoice(0); math.Abs(got-want) > 1e-8 {
		t.Fatalf("zero-period resume: %g; want %g", got, want)
	}
}

func TestRestoringPitchReschedulesEvenBeforePendingValueExpires(t *testing.T) {
	p := &Player{data: []byte{10, 70, 110}}
	p.streams[0] = streamVoice{Start: 0, Length: 3, LoopStart: 0, LoopLength: 3, Period: 428, Volume: 64, Starts: 1, Enabled: true}
	p.mixVoice(0)
	p.streams[0].Period = 0
	p.mixVoice(0)
	p.streams[0].Period = 213
	want := float64(10*SampleRate+70*(pitchClock-SampleRate)) * 128 / float64(pitchClock)
	if got := p.mixVoice(0); math.Abs(got-want) > 1e-8 {
		t.Fatalf("early pitch restore: %g; want %g", got, want)
	}
}
