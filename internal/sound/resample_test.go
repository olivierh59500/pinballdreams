package sound

import (
	"math"
	"sort"
	"testing"
)

const referencePitchClock int64 = 3546895

func TestPitchChangeKeepsTheAlreadyScheduledValue(t *testing.T) {
	p := &Player{data: []byte{0, 100}}
	p.streams[0] = streamVoice{Start: 0, Length: 2, LoopStart: 0, LoopLength: 2, Period: 428, Volume: 64, Starts: 1, Enabled: true}
	if p.mixVoice(0) != 0 {
		t.Fatal("unexpected initial value")
	}
	p.streams[0].Period = 113
	for frame := 1; frame < 5; frame++ {
		if got := p.mixVoice(0); got != 0 {
			t.Fatalf("pitch change shortened the held value at frame%d: %g", frame, got)
		}
	}
	want := float64(6*referencePitchClock-428*SampleRate) * 100 * 128 / float64(referencePitchClock)
	if got := p.mixVoice(0); math.Abs(got-want) > 1e-8 {
		t.Fatalf("pending interval boundary: native%g expected%g", got, want)
	}
}

type pitchChange struct {
	frame  int
	period uint16
}

type heldValue struct {
	at, area int64
	value    int64
}

// referenceHeldAudio builds the entire source-value timeline first, then
// samples its prefix integral. It is independent of the production streaming
// cursor. Synthetic byte values are test inputs, never game resources.
func referenceHeldAudio(data []byte, stream streamVoice, changes []pitchChange, frames int) []float64 {
	timeline := []heldValue{}
	period, change := stream.Period, 0
	var at, area int64
	for index := 0; at < int64(frames)*referencePitchClock; index++ {
		for change < len(changes) && int64(changes[change].frame)*referencePitchClock <= at {
			period = changes[change].period
			change++
		}
		position := stream.Start + index
		if index >= stream.Length {
			position = stream.LoopStart + (index-stream.Length)%stream.LoopLength
		}
		value := int64(int8(data[position]))
		timeline = append(timeline, heldValue{at: at, area: area, value: value})
		duration := int64(period) * SampleRate
		area += value * duration
		at += duration
	}
	integral := func(at int64) int64 {
		i := sort.Search(len(timeline), func(i int) bool { return timeline[i].at > at }) - 1
		return timeline[i].area + (at-timeline[i].at)*timeline[i].value
	}
	result := make([]float64, frames)
	for frame := range result {
		begin, end := int64(frame)*referencePitchClock, int64(frame+1)*referencePitchClock
		result[frame] = float64(integral(end)-integral(begin)) * float64(stream.Volume) * 2 / float64(referencePitchClock)
	}
	return result
}

func TestVoiceIntegratesHeldValuesAndKeepsPendingPitchInterval(t *testing.T) {
	data := []byte{0, 127, 128, 30, 212, 70, 0, 100, 23, 193, 64}
	for _, period := range []uint16{1, 113, 127, 213, 428, 856, 65535} {
		for _, variable := range []bool{false, true} {
			stream := streamVoice{Start: 0, Length: 7, LoopStart: 2, LoopLength: 5, Period: period, Volume: 64, Starts: 1, Enabled: true}
			var changes []pitchChange
			if variable {
				changes = []pitchChange{{1, 113}, {37, 856}, {41, 127}, {719, period}}
			}
			want := referenceHeldAudio(data, stream, changes, 2048)
			p := &Player{data: append([]byte(nil), data...)}
			p.streams[0] = stream
			change := 0
			for n, expected := range want {
				if change < len(changes) && changes[change].frame == n {
					p.streams[0].Period = changes[change].period
					change++
				}
				got := p.mixVoice(0)
				if math.Abs(got-expected) > 1e-8 {
					t.Fatalf("period%d variable%t frame%d: native%g integrated%g", period, variable, n, got, expected)
				}
			}
		}
	}
}

func TestVoiceRetriggerOnlyRestartsItsOwnSegment(t *testing.T) {
	data := []byte{10, 70, 128, 45, 23, 210, 127, 44, 30, 80, 190}
	stream := streamVoice{Start: 0, Length: 7, LoopStart: 2, LoopLength: 5, Period: 428, Volume: 64, Starts: 1, Enabled: true}
	p := &Player{data: data}
	p.streams[0], p.streams[1] = stream, stream
	wantOther := referenceHeldAudio(data, stream, nil, 256)
	for i := 0; i < 33; i++ {
		p.mixVoice(0)
		p.mixVoice(1)
	}
	stream.Start, stream.Length, stream.LoopStart, stream.LoopLength, stream.Period, stream.Starts = 7, 4, 7, 4, 213, 2
	p.streams[0] = stream
	wantRestart := referenceHeldAudio(data, stream, nil, 128)
	for i, want := range wantRestart {
		if got := p.mixVoice(0); math.Abs(got-want) > 1e-8 {
			t.Fatalf("retrigger frame%d: %g !=%g", i, got, want)
		}
		if got := p.mixVoice(1); math.Abs(got-wantOther[i+33]) > 1e-8 {
			t.Fatal("retrigger moved another voice")
		}
	}
}

func TestLoopDeclarationWaitsForCurrentSegment(t *testing.T) {
	data := []byte{10, 70, 128, 45, 23, 210, 127, 44, 30, 80, 190}
	stream := streamVoice{Start: 0, Length: 7, LoopStart: 2, LoopLength: 5, Period: 428, Volume: 64, Starts: 1, Enabled: true}
	p := &Player{data: data}
	p.streams[0] = stream
	// The new loop is declared before the seven-value initial segment ends;
	// its first sample cannot replace any remaining initial-segment value.
	stream.LoopStart, stream.LoopLength = 7, 4
	want := referenceHeldAudio(data, stream, nil, 256)
	for i, w := range want {
		if i == 3 {
			p.streams[0].LoopStart, p.streams[0].LoopLength = 7, 4
		}
		if got := p.mixVoice(0); math.Abs(got-w) > 1e-8 {
			t.Fatalf("loop update frame%d: %g !=%g", i, got, w)
		}
	}
}

func TestHeldValueSurvivesPrivateBufferMutation(t *testing.T) {
	p := &Player{data: []byte{50, 70}}
	p.streams[0] = streamVoice{Start: 0, Length: 2, LoopStart: 0, LoopLength: 2, Period: 856, Volume: 64, Starts: 1, Enabled: true}
	p.mixVoice(0)
	p.data[0] = 100
	for n := 0; n < 8; n++ {
		if got := p.mixVoice(0); got != 50*64*2 {
			t.Fatalf("buffer mutation replaced an in-progress held value: %g", got)
		}
	}
}
