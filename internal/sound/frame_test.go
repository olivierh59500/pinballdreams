package sound

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestGameClockPCMMatchesStandaloneMusicCuesFadesAndPause(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			module := sourceModule(t, id)
			clocked, reference := New(module, -1), New(module, -1)
			var state OutputState
			got, want := make([]byte, FrameBytes), make([]byte, FrameBytes)
			for frame := range 600 {
				for _, player := range []*Player{clocked, reference} {
					if frame%91 == 0 {
						player.Flipper()
					}
					if frame%131 == 31 {
						player.ReleaseSpring(frame % 16)
					}
					if frame%191 == 79 {
						player.BeginPlay(true)
					}
					if frame%191 == 139 {
						player.BeginPlay(false)
					}
					if frame == 440 {
						player.Fade(0, 1)
					}
					if frame == 520 {
						player.Fade(64, 1)
					}
					player.Pause(frame >= 240 && frame < 260)
					player.Mute(frame >= 360 && frame < 390)
				}
				clocked.AdvanceFrame()
				clocked.RenderFrame(got, &state)
				if _, err := reference.Read(want); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("game-clock PCM differs at frame %d", frame)
				}
			}
		})
	}
}

func TestOutputFilterTailSurvivesChangingPlayers(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			module := sourceModule(t, id)
			first := New(module, -1)
			var output OutputState
			block := make([]byte, FrameBytes)
			found := false
			for frame := 0; frame < 200 && !found; frame++ {
				first.AdvanceFrame()
				first.RenderFrame(block, &output)
				found = binary.LittleEndian.Uint16(block[len(block)-4:]) != 0 || binary.LittleEndian.Uint16(block[len(block)-2:]) != 0
			}
			if !found {
				t.Fatal("original song did not leave an audible output tail")
			}
			quiet, reset := New(module, module.QuietMusic.Order), New(module, module.QuietMusic.Order)
			quiet.SetLowPass(first.LowPassEnabled())
			reset.SetLowPass(first.LowPassEnabled())
			quiet.AdvanceFrame()
			reset.AdvanceFrame()
			got, fresh := make([]byte, FrameBytes), make([]byte, FrameBytes)
			quiet.RenderFrame(got, &output)
			reset.RenderFrame(fresh, &OutputState{})
			if !bytes.Equal(fresh, make([]byte, FrameBytes)) {
				t.Fatal("fresh original quiet song is not silent")
			}
			if bytes.Equal(got, fresh) {
				t.Fatal("module replacement discarded the previous output tail")
			}
		})
	}
}

func TestPCMQueueUsesOnlyCopiedFramesAndPreservesPartialReads(t *testing.T) {
	module, err := DecodeFrontend(assets.Files(), false)
	if err != nil {
		t.Fatal(err)
	}
	player, queue := New(module, -1), NewPCMQueue(0)
	var output OutputState
	block := make([]byte, FrameBytes)
	var want []byte
	for range 24 {
		player.AdvanceFrame()
		player.RenderFrame(block, &output)
		want = append(want, block...)
		queue.Push(block)
	}
	clear(block)
	var got []byte
	for n := 0; queue.BufferedBytes() != 0; n++ {
		part := make([]byte, []int{1, 7, 511, 4096, 19200}[n%5])
		count, err := queue.Read(part)
		if err != nil || count == 0 || count > FrameBytes {
			t.Fatalf("queued read: %d, %v", count, err)
		}
		got = append(got, part[:count]...)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("queue changed PCM or retained a producer buffer alias")
	}
	if n, err := queue.Read(block); n != 0 || err != nil {
		t.Fatalf("empty queue ended the stream: %d, %v", n, err)
	}
}

func TestFractionalDurationPreservesPCMThroughArbitraryStereoBlocks(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			module := sourceModule(t, table)
			whole, split := New(module, -1), New(module, -1)
			var wholeState, splitState OutputState
			for frame := range 100 {
				for _, player := range []*Player{whole, split} {
					if frame == 25 {
						player.Flipper()
					}
					player.AdvanceFrame()
				}
				got, want := make([]byte, (961+frame%2)*4), make([]byte, (961+frame%2)*4)
				whole.RenderFrames(want, &wholeState)
				for at := 0; at < len(got); {
					end := min(at+(1+frame%31)*4, len(got))
					split.RenderFrames(got[at:end], &splitState)
					at = end
				}
				if !bytes.Equal(got, want) || whole.OutputDeclarations() != split.OutputDeclarations() {
					t.Fatalf("block boundaries changed PCM or replay at step %d", frame)
				}
			}
		})
	}
}
