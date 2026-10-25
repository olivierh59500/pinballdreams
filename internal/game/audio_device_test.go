package game

import (
	"errors"
	"testing"
	"time"

	"github.com/olivierh59500/pinballdreams/internal/sound"
)

type focusDeviceProbe struct {
	suspends, resumes int
	err               error
}

func (d *focusDeviceProbe) Suspend() error { d.suspends++; return d.err }
func (d *focusDeviceProbe) Resume() error  { d.resumes++; return d.err }

func TestDeviceFocusTransitionsAreIdempotentAndExposeErrors(t *testing.T) {
	device := &focusDeviceProbe{}
	state := audioFocusState{active: true}
	for range 8 {
		if err := state.update(false, device); err != nil {
			t.Fatal(err)
		}
	}
	if state.active || device.suspends != 1 || device.resumes != 0 {
		t.Fatal("background transition repeated or remained active")
	}
	if err := state.update(true, device); err != nil {
		t.Fatal(err)
	}
	if !state.active || device.resumes != 1 {
		t.Fatal("foreground did not resume its device")
	}
	want := errors.New("device unavailable")
	device.err = want
	if err := state.update(false, device); !errors.Is(err, want) || !state.active {
		t.Fatal("failed suspension was hidden or recorded as successful")
	}
}

func TestMacOutputBudgetCoversPrimingAndSchedulingMargin(t *testing.T) {
	timing := timingForPlatform("darwin")
	const tick = time.Second / sound.TickRate
	if timing.device <= 0 || timing.device > 20*time.Millisecond || timing.player < timing.device/4 {
		t.Fatal("macOS device callback no longer fits its tested source buffer")
	}
	reserve := time.Duration(timing.reserveFrames) * tick
	if reserve > 40*time.Millisecond || reserve < timing.device+16*time.Millisecond {
		t.Fatal("macOS reserve misses the latency bound or tested jitter margin")
	}
	android := timingForPlatform("android")
	if android.player < 100*time.Millisecond || time.Duration(android.reserveFrames)*tick < 140*time.Millisecond {
		t.Fatal("unmeasured Android burst sizes lost their conservative budget")
	}
}
