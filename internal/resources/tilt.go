package resources

import "github.com/olivierh59500/pinballdreams/internal/audiodata"

// TiltPresentation contains the supplied nudge limits, text and music requests.
// A zero warning meter means that the original table declares no warning cue.
type TiltPresentation struct {
	WarningMeter, TiltMeter, DecayFrames, InitialDecay int
	Warning, EntryCue                                  audiodata.Command
	Text                                               string
}
