package sound

// Effect names follow the tracker's musical operations. Their numeric values
// remain part of the extracted note data, with no executable command stream.
const (
	effectArpeggio = iota
	effectPitchUp
	effectPitchDown
	effectPortamento
	effectVibrato
	effectPortamentoVolume
	effectVibratoVolume
	effectTremolo
	effectUnusedPan
	effectSampleOffset
	effectVolumeSlide
	effectPositionJump
	effectSetVolume
	effectPatternBreak
	effectExtended
	effectSpeed
)
