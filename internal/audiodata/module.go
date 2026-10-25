// Package audiodata defines native tracker notes, sample ranges and audio cues.
// Original packing and executable layouts are handled only by the extractor.
package audiodata

type Note struct{ Sample, Pitch, Effect, Parameter int }

type Sample struct {
	Offset, Length                          int
	Volume, Finetune, LoopStart, LoopLength int
}

type Action string

const (
	None  Action = "none"
	Sound Action = "note"
	Music Action = "music"
)

// Command is a decoded game sound or music selection. Only music selections
// use Priority; notes and empty commands leave it zero.
type Command struct {
	Action   Action
	Note     Note
	Order    int
	Repeats  byte
	Priority uint16
}

// SpringSound scales the supplied release sample's volume by power level.
type SpringSound struct {
	Cue                     Command
	VolumeStep, VolumeLimit int
}

func (s SpringSound) ReleaseCue(level int) Command {
	volume := min(s.VolumeLimit, max(0, level)*s.VolumeStep)
	if volume == 0 {
		return Command{Action: None}
	}
	command := s.Cue
	command.Note.Parameter = volume
	return command
}

// Provenance supports development comparisons. The native player does not use
// source addresses or compressed byte positions for playback or sequencing.
type Provenance struct {
	PCMAddress     int
	RowByteOffsets [][65]uint16
}

type Module struct {
	Version               int
	SourceSHA256          string
	Samples               []Sample
	PCM                   []byte
	Orders                []int
	Patterns              [][64][4]Note
	Periods               [16][36]int
	PeriodLookup          [577]uint16
	Wave                  [32]byte
	FunkRates             [16]byte
	InitialOrder          int
	BallStart             []Command
	PlayMusic, QuietMusic Command
	PlaySensor            int
	InitialMusicDisabled  bool
	Flipper               Command
	SpringRelease         SpringSound
	Provenance            Provenance
}

// Period includes the supplied preceding entry used by zero-based effects.
// Stored notes use pitch indices one through thirty-six.
func (m *Module) Period(tune byte, pitch int) uint16 {
	index := int(tune&15)*36 + pitch
	if index < 0 || index >= len(m.PeriodLookup) {
		return 0
	}
	return m.PeriodLookup[index]
}
