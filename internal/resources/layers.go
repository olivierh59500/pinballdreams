package resources

// Region is an ordered source rectangle and the data object it selects.
type Region struct{ X0, Y0, X1, Y1, Object int }

// Layer retains a table level's collision, sprite mask and interaction records.
// The source switches these records together when the ball enters a ramp gate.
type Layer struct {
	Collision                          []byte
	CollisionAddress, OcclusionAddress int
	Kickers                            []Kicker
	Sensors, ExtraSensors              []Sensor
	Targets                            []Region
	References                         []int
	Hidden                             []byte
	HiddenWidth, HiddenHeight          int
	GravityEdges                       []GravityProbe
}

// LayerGate selects a level using one of the source's two sentinel identities.
type LayerGate struct {
	Region
	Layer int
}

// CollisionOverlap links native cells shared by two levels. A target update
// in either view must also update the matching cells in the other view.
type CollisionOverlap struct {
	FromLayer, ToLayer, FromIndex, ToIndex, Length int
}

// SelectLayer preserves the source's ordered gate traversal and early Y exit.
func (t *Table) SelectLayer(current, x, y int) int {
	for _, gate := range t.LayerGates {
		if x < gate.X0 || x > gate.X1 {
			continue
		}
		if y < gate.Y0 {
			return current
		}
		if y > gate.Y1 {
			continue
		}
		return gate.Layer
	}
	return current
}

// BallHiddenOnLayer reads the selected level's original sprite occlusion mask.
func (t *Table) BallHiddenOnLayer(level, x, y int) bool {
	layer := &t.Layers[level]
	width, height := layer.HiddenWidth, layer.HiddenHeight
	if width == 0 {
		width, height = 320, len(layer.Hidden)/320
	}
	if x < 0 || x >= width || y < 0 || y >= height {
		return false
	}
	return layer.Hidden[y*width+x] != 0
}

// MaterialSpan samples one channel of eight adjacent collision cells. Edge
// probes use these extracted spans to preserve the supplied field boundaries.
type MaterialSpan struct {
	X, Y     int
	Mask     byte
	Constant byte
	Dynamic  bool
}

type GravityProbe struct {
	Clearance [2]MaterialSpan
	Selection MaterialSpan
}
