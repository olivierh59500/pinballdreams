package timing

import "testing"

func TestTableHostAndSamplesShareElapsedTimeWithoutDrift(t *testing.T) {
	var host HostClock
	var samples Samples
	const polls = 10000 * HostTicksPerSecond
	steps, frames := 0, 0
	for range polls {
		if !host.Tick(Table) {
			continue
		}
		steps++
		block := samples.Next(Table, 48000)
		if block != 961 && block != 962 {
			t.Fatalf("unexpected table block length %d", block)
		}
		frames += block
	}
	// The initial step is immediate. Later steps must stay within one game
	// interval of wall time over this 10000-second deterministic run.
	wantSteps := 1 + int64(polls-1)*3546895/(71051*HostTicksPerSecond)
	if int64(steps) != wantSteps || int64(frames) != int64(steps)*48000*71051/3546895 {
		t.Fatalf("steps=%d frames=%d", steps, frames)
	}
	if frames < 48000*polls/HostTicksPerSecond-962 || frames > 48000*polls/HostTicksPerSecond+962 {
		t.Fatal("PCM and host elapsed time drifted apart")
	}
}

func TestNominalCadenceAndTransitions(t *testing.T) {
	var host HostClock
	var samples Samples
	for poll := range HostTicksPerSecond {
		if got := host.Tick(Nominal); got != (poll%4 == 0) {
			t.Fatalf("nominal poll %d", poll)
		}
		if samples.Next(Nominal, 48000) != 960 {
			t.Fatal("nominal audio duration changed")
		}
	}
	for _, cadence := range []Cadence{Table, Nominal, Table} {
		if got := samples.Next(cadence, 48000); got > cadence.MaxSamples(48000) || got < 960 {
			t.Fatalf("scene transition audio block %d", got)
		}
	}
}

func TestSceneChangesPreserveElapsedTimeAndFractionalAudio(t *testing.T) {
	var host HostClock
	if !host.Tick(Nominal) {
		t.Fatal("initial scene did not start")
	}
	for _, cadence := range []Cadence{Table, Nominal, Table, Table} {
		if host.Tick(cadence) {
			t.Fatal("a scene change produced an extra game step")
		}
	}
	if !host.Tick(Nominal) {
		t.Fatal("scene transition lost accumulated elapsed time")
	}
	var samples Samples
	frames := 0
	for range 10000 {
		frames += samples.Next(Table, 48000)
		frames += samples.Next(Nominal, 48000)
	}
	want := int64(10000)*48000*71051/3546895 + 10000*960
	if int64(frames) != want {
		t.Fatalf("scene transitions discarded fractional samples: got%d want%d", frames, want)
	}
}
