// Package resources reconstructs game images and numerical tables from ADF data.
// Original artwork, fonts and audio are supplied by the bundled native resources.
package resources

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"os"

	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

const Base = 0x3000

// Frontend records retain the extractor's source-offset and load-address names.
const IntroFile = "disk1-002c9e-002ff8.bin"
const SelectionFile = "disk1-046c54-002ff8.bin"
const IntroMusicFile = "disk1-0162f6-040000.bin"
const MenuMusicFile = "disk1-052dfe-03fff8.bin"

var Names = [...]string{"Ignition", "Steel Wheel", "Beat Box", "Nightmare"}
var Files = [...]string{"disk2-00049e-002ff8.bin", "disk2-039558-002ff8.bin", "disk2-06b2de-002ff8.bin", "disk2-0a09e8-002ff8.bin"}

// RecordHashes identify the four reconstructed records from the supported ADF
// pair. Native bundles must agree with these extraction provenance hashes.
var RecordHashes = [...]string{
	"93a68d7f348b9e42d44de11edf8867e67e817cc3a6f83df40df5d52dfe957446",
	"ce62c62a7d6f7e0ae3b99fac7b9dd3067d8b36961229f5e1f808fdd72abe3ff1",
	"65e15617c141c3d849ebb0d3c1c63e9eb2ce3ab139a134d2d4494b72973ff79f",
	"73631bd0a4bad15e92cc7a0842c20753742a9e6a9190cdaab0d326c35cc1f192",
}

// Table contains immutable source data and its reconstructed table artwork.
type Table struct {
	ScoreSeparators                 ScoreSeparators
	ID                              int
	Data                            []byte `json:"-"`
	Art                             *image.Paletted
	Ball                            *image.Paletted
	Header                          *image.Paletted
	ScoreLabel                      string
	ScorePlayerOffset               int
	Flippers                        []Flipper
	Collision                       []byte
	Layers                          []Layer
	LayerGates                      []LayerGate
	CollisionOverlaps               []CollisionOverlap
	SensorFreezeGuard               bool
	TargetStanding, TargetDropped   int
	OcclusionAddress                int
	Points                          [44]Point
	ProbeOrder                      []int
	Sine, Cosine                    [2048]int16
	Materials                       [8]Material
	ContactRoutine, VelocityAddress int
	MusicAddress                    int
	Kickers                         []Kicker
	Font                            [224]*image.Paletted
	Gravity                         [128][2]int16
	GravityOffset                   int
	GravityDefault                  [2]int16
	SpringStep, SpringLimit         int
	SpringMouseScale                int
	SpringQuantized                 bool
	Sensors                         []Sensor
	Rules                           *ruledata.Definitions `json:"-"`
	Audio                           *audiodata.Module     `json:"-"`
	TargetPatches                   map[int][]CollisionChange
	ScoreEntry                      ScoreEntryUI
	BallPreparation                 BallPreparationUI
	Launch                          LaunchDefinition
	Tilt                            TiltPresentation
	Quit                            QuitUI
	PauseText                       string
	Attract                         TableAttract
	Viewport                        TableViewport
	SpringGauge                     SpringGauge `json:"-"`
	RolloverColors                  [64]bool
}

type ScoreEntryUI struct {
	Prompt, Allowed                     string
	PlayerCell, InitialsCell            int
	Music                               audiodata.Command
	GameOverText                        string
	GameOverMusic                       audiodata.Command
	GameOverDelay, PlayerDelay, Repeats int
	GameOverLoopBytes                   int
}

// BallPreparationUI contains the supplied pre-launch countdown and status line.
type BallPreparationUI struct {
	HoldFrames            [2]int
	Prompt                string
	BallCell, BlinkFrames int
	SharedBlink           bool
	ScoreAlternateFrames  int
	Cues                  [2]audiodata.Command
	Reserved, Relaunch    PlayerPrompt
	BlankPrompt           string
	SpecialBlinkFrames    int
}

// PlayerPrompt stores an original status template and its writable player cell.
type PlayerPrompt struct {
	Text       string
	PlayerCell int
}

// LaunchDefinition names the source entry latches that control spring play.
type LaunchDefinition struct {
	SpringSensor, PlaySensor int
	CancelCue                audiodata.Command
}

// SpringGauge contains supplied table pixels for each spring-power level.
type SpringGauge struct {
	X, Y, UnitsPerLevel int
	Frames              []*image.Paletted
}

func (g *SpringGauge) Level(charge int) int {
	return max(0, min(len(g.Frames)-1, charge/g.UnitsPerLevel))
}

// Sensor is an original rectangular rollover, lane or trap detector.
type Sensor struct{ X0, Y0, X1, Y1, Address, Kind int }

// Kicker bounds identify the original powered bumpers and slingshots.
type Kicker struct{ X0, Y0, X1, Y1, Material, Address int }

// Point is one of the original 44 ball perimeter probes, in source order.
type Point struct{ X, Y, Angle int }

// Flipper retains every original animation frame and its collision silhouette.
type Flipper struct {
	Right                bool
	X, Y, PivotX, PivotY int
	Art                  [17]*image.Paletted
	Mask                 [17][]bool
}

// Load reads generated records. The extractor is the only producer of these files.
func Load(directory string, id int) (*Table, error) {
	return LoadFS(os.DirFS(directory), id)
}

// LoadFS supports the generated embedded bundle used by desktop and Android.
func LoadFS(source fs.FS, id int) (*Table, error) {
	if id < 0 || id >= len(Files) {
		return nil, fmt.Errorf("invalid table %d", id)
	}
	t, err := loadNativeTable(source, id, RecordHashes[id])
	if err != nil {
		return nil, err
	}
	t.Rules, err = ruledata.LoadNativeFS(source, id, RecordHashes[id])
	if err != nil {
		return nil, err
	}
	t.Audio, err = audiodata.LoadFS(source, audiodata.TableFilename(id), RecordHashes[id])
	return t, err
}

// LoadSourceFS is a development-only path for source-data comparison tools.
// Applications use LoadFS and never request the reconstructed executables.
func LoadSourceFS(source fs.FS, id int) (*Table, error) {
	t, err := LoadFS(source, id)
	if err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(source, Files[id])
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != RecordHashes[id] {
		return nil, fmt.Errorf("source comparison record does not match table %d", id+1)
	}
	t.Data = data
	return t, nil
}

// Material names the integer response parameters extracted for a surface.
// Divisors retain the artwork's original tuning and fixed-point precision.
type Material struct {
	TangentDrag, SpinDrag, BounceDivisor int16
	NormalLimit, SlopeLimit              int16
	KickThreshold, KickImpulse           int16
}

// CollisionChange replaces selected material bits in one native atlas cell.
type CollisionChange struct {
	Index                  int
	Mask, Standing, Active byte
}

// Word reads a big-endian source data word using the original load address.
func (t *Table) Word(address int) uint16 { return binary.BigEndian.Uint16(t.Data[address-Base:]) }

// Long reads a big-endian source data long using the original load address.
func (t *Table) Long(address int) uint32 { return binary.BigEndian.Uint32(t.Data[address-Base:]) }

// SpringImpulse preserves the supplied Ignition release's high-byte-only copy.
// The other tables copy the complete signed spring word.
func (t *Table) SpringImpulse(charge int) int {
	if t.SpringQuantized {
		return -int(int16(uint16(-charge) & 0xff00))
	}
	return charge
}

// ScoreText uses the original ten-character player label and right-hand ten
// decimal digits. The source blanks leading zeroes in its score renderer.
func (t *Table) ScoreText(player int, score uint64) string {
	label := []byte(t.ScoreLabel)
	label[t.ScorePlayerOffset] = byte('1' + player)
	return string(label) + fmt.Sprintf("%10d", score%10_000_000_000)
}

// BallHidden reads the source sprite occlusion bit at a table-space pixel.
func (t *Table) BallHidden(x, y int) bool {
	return t.BallHiddenOnLayer(0, x, y)
}

// Planar converts scanline-interleaved bitplanes directly to palette indices.
func Planar(data []byte, width, height, planes int, palette color.Palette) (*image.Paletted, error) {
	if width <= 0 || width%8 != 0 || height <= 0 || planes < 1 || planes > 8 || len(palette) < 1<<planes {
		return nil, fmt.Errorf("invalid planar bitmap geometry")
	}
	stride := width / 8
	if len(data) < stride*height*planes {
		return nil, fmt.Errorf("truncated planar bitmap")
	}
	im := image.NewPaletted(image.Rect(0, 0, width, height), palette)
	for y := range height {
		for x := range width {
			var index byte
			for plane := range planes {
				index |= ((data[(y*planes+plane)*stride+x/8] >> uint(7-x%8)) & 1) << uint(plane)
			}
			im.Pix[y*im.Stride+x] = index
		}
	}
	return im, nil
}
