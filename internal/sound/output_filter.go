package sound

import "math"

// These response corners define the supported original-machine listening
// profile. They shape ordinary mixed PCM; no audio device is simulated.
var outputFilterWeights = [...]float64{
	lowPassWeight(6200), lowPassWeight(20000), lowPassWeight(7000),
}

func lowPassWeight(corner float64) float64 {
	prewarped := 2 * math.Tan(math.Pi*corner/SampleRate)
	return prewarped / (1 + prewarped)
}

type outputFilter struct {
	fixed [2]float64
	tone  [3]float64
}

func (f *outputFilter) sample(input float64, toneEnabled bool) float64 {
	for n := range f.fixed {
		f.fixed[n] += outputFilterWeights[n] * (input - f.fixed[n])
		input = f.fixed[n]
	}
	normal := input
	// Both responses retain history even while the other is selected. E0x
	// changes therefore preserve the tail of the supplied audio waveform.
	for n := range f.tone {
		f.tone[n] += outputFilterWeights[2] * (input - f.tone[n])
		input = f.tone[n]
	}
	if toneEnabled {
		return input
	}
	return normal
}

// filterSample applies the native output tone stage independently on each
// stereo side. Tracker commands only select the optional tone response.
func (p *Player) filterSample(left, right float64) (float64, float64) {
	return p.outputFilters[0].sample(left, p.filter), p.outputFilters[1].sample(right, p.filter)
}
