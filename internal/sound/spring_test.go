package sound

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestSpringReleaseStartsTheSuppliedEventVoiceAtItsPowerVolume(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			module := sourceModule(t, id)
			player := New(module, -1)
			var commands []audiodata.Command
			player.ObserveCommands(func(command Command) { commands = append(commands, command) })
			player.ReleaseSpring(0)
			if len(commands) != 0 || player.streams[3].Starts != 0 {
				t.Fatal("a sub-step release requested an event sample")
			}
			for n, level := range []int{1, 8, 16} {
				player.ReleaseSpring(level)
				voice, stream := player.channels[3], player.streams[3]
				if len(commands) != n+1 || !stream.Enabled || stream.Starts != uint32(n+1) || voice.volume != byte(level*4) {
					t.Fatalf("release at level %d did not start one source-volume sample: voice=%+v stream=%+v", level, voice, stream)
				}
				if stream.Start != module.Samples[module.SpringRelease.Cue.Note.Sample-1].Offset || voice.pitch != uint16(module.SpringRelease.Cue.Note.Pitch) {
					t.Fatal("spring release selected a substitute sample or pitch")
				}
			}
		})
	}
}

func TestCommandObserverCanUnsubscribeOutsidePlaybackMutex(t *testing.T) {
	player := New(sourceModule(t, 0), -1)
	calls := 0
	player.ObserveCommands(func(Command) {
		calls++
		player.ObserveCommands(nil)
	})
	player.ReleaseSpring(8)
	player.ReleaseSpring(16)
	if calls != 1 {
		t.Fatal("command observer did not unsubscribe after its first callback")
	}
}
