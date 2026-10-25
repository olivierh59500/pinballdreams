// Package timing keeps native game steps and PCM duration on the same clock.
package timing

// Input polls do not run physics or replay. Five-millisecond polls retain
// short actions while the heavier table update keeps its original cadence.
const HostTicksPerSecond = 200

const (
	nominalRate          int64 = 50
	tableRateNumerator   int64 = 3546895
	tableRateDenominator int64 = 71051
)

type Cadence uint8

const (
	Nominal Cadence = iota
	Table
)

func (c Cadence) ratio() (numerator, denominator int64) {
	if c == Table {
		// Preserve the observed update rate as a fraction, about 49.9204 Hz.
		return tableRateNumerator, tableRateDenominator
	}
	return nominalRate, 1
}

// HostClock samples input more often than a game step. Integer accumulation
// retains the table's fractional cadence without rounding it to 50 Hz.
type HostClock struct {
	phase int64
	ready bool
}

func (c *HostClock) Tick(cadence Cadence) bool {
	// Both scene rates share this time grid. A scene transition preserves
	// elapsed time instead of creating an extra update and queued audio block.
	const pollUnits = tableRateNumerator
	period := pollUnits * HostTicksPerSecond / nominalRate
	if cadence == Table {
		period = tableRateDenominator * HostTicksPerSecond
	}
	if !c.ready {
		c.ready = true
		c.phase = period - pollUnits
	}
	c.phase += pollUnits
	if c.phase < period {
		return false
	}
	c.phase -= period
	return true
}

// Samples carries fractional stereo-frame counts across successive game steps.
// Its output and HostClock therefore represent the same elapsed game time.
type Samples struct {
	phase int64
}

func (s *Samples) Next(cadence Cadence, sampleRate int) int {
	const unitsPerSecond = tableRateNumerator * nominalRate
	duration := tableRateNumerator
	if cadence == Table {
		duration = tableRateDenominator * nominalRate
	}
	s.phase += int64(sampleRate) * duration
	frames := s.phase / unitsPerSecond
	s.phase %= unitsPerSecond
	return int(frames)
}

func (c Cadence) MaxSamples(sampleRate int) int {
	numerator, denominator := c.ratio()
	return int((int64(sampleRate)*denominator + numerator - 1) / numerator)
}
