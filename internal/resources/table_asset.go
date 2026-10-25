package resources

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io/fs"
)

func TableAssetFilename(id int) string { return fmt.Sprintf("table%d-art-physics.json", id+1) }

// IndexedImage stores ordinary pixel indices and fully resolved RGBA colors.
// Original planar layouts and sprite row padding are resolved by extraction.
type IndexedImage struct {
	Width, Height int
	Pixels        []byte
	Colors        []color.RGBA
}

func (p *IndexedImage) Image() *image.Paletted {
	if p.Width == 0 {
		return nil
	}
	palette := make(color.Palette, len(p.Colors))
	for n, c := range p.Colors {
		palette[n] = c
	}
	return &image.Paletted{Pix: p.Pixels, Stride: p.Width, Rect: image.Rect(0, 0, p.Width, p.Height), Palette: palette}
}

type FlipperAsset struct {
	Right                bool
	X, Y, PivotX, PivotY int
	Art                  [17]IndexedImage
	Mask                 [17][]byte
}

type SpringGaugeAsset struct {
	X, Y, UnitsPerLevel int
	Frames              []IndexedImage
}

// TableAsset adds a native image encoding to the table's numerical data.
// The named image fields shadow their image.Paletted runtime counterparts.
// Raw executable bytes and mutable rule state are never serialized here.
type TableAsset struct {
	Table
	Version           int
	SourceSHA256      string
	Art, Ball, Header IndexedImage
	Flippers          []FlipperAsset
	Font              [224]IndexedImage
	SpringGauge       SpringGaugeAsset
}

func loadNativeTable(source fs.FS, id int, expectedSHA256 string) (*Table, error) {
	data, err := fs.ReadFile(source, TableAssetFilename(id))
	if err != nil {
		return nil, fmt.Errorf("extract native table artwork and geometry first: %w", err)
	}
	var a TableAsset
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("native table assets: %w", err)
	}
	if a.ID != id || a.SourceSHA256 != expectedSHA256 {
		return nil, fmt.Errorf("native artwork and geometry do not match table %d", id+1)
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	t := &a.Table
	t.Art, t.Ball, t.Header = a.Art.Image(), a.Ball.Image(), a.Header.Image()
	for _, decoded := range a.Flippers {
		f := Flipper{Right: decoded.Right, X: decoded.X, Y: decoded.Y, PivotX: decoded.PivotX, PivotY: decoded.PivotY}
		for frame := range f.Art {
			f.Art[frame] = decoded.Art[frame].Image()
			f.Mask[frame] = make([]bool, len(decoded.Mask[frame]))
			for p, value := range decoded.Mask[frame] {
				f.Mask[frame][p] = value != 0
			}
		}
		t.Flippers = append(t.Flippers, f)
	}
	for n := range t.Font {
		t.Font[n] = a.Font[n].Image()
	}
	t.SpringGauge = SpringGauge{X: a.SpringGauge.X, Y: a.SpringGauge.Y, UnitsPerLevel: a.SpringGauge.UnitsPerLevel}
	for _, frame := range a.SpringGauge.Frames {
		t.SpringGauge.Frames = append(t.SpringGauge.Frames, frame.Image())
	}
	return t, nil
}

// Validate rejects malformed native image/geometry bundles before gameplay.
func (a *TableAsset) Validate() error {
	if a.Version != 19 || a.ID < 0 || a.ID > 3 || len(a.SourceSHA256) != 64 || len(a.Layers) != 2 || len(a.Flippers) != 2 || a.TargetPatches == nil || len(a.ProbeOrder) == 0 || a.SpringMouseScale != 7 || len(a.PauseText) != 20 || a.Launch.CancelCue.Action == "" || a.Launch.CancelCue.Action == "none" {
		return fmt.Errorf("invalid native table schema")
	}
	if err := a.Viewport.Validate(); err != nil {
		return err
	}
	if a.ScoreSeparators.Digits < 1 || a.ScoreSeparators.LastCell < 0 || a.ScoreSeparators.LastCell >= 20 || a.ScoreSeparators.Y < 19 || len(a.ScoreSeparators.Pixels) == 0 {
		return fmt.Errorf("invalid native score separators")
	}
	if err := a.Attract.Validate(); err != nil {
		return err
	}
	classified := false
	for _, contact := range a.RolloverColors {
		classified = classified || contact
	}
	if !classified {
		return fmt.Errorf("missing native rollover color classification")
	}
	preparation := a.BallPreparation
	if preparation.HoldFrames[0] < 1 || preparation.HoldFrames[1] < 1 || len(preparation.Prompt) != 20 || preparation.BallCell < 8 || preparation.BallCell >= 20 || preparation.BlinkFrames < 1 {
		return fmt.Errorf("invalid ball preparation presentation")
	}
	if len(preparation.BlankPrompt) != 20 || preparation.SpecialBlinkFrames < 1 {
		return fmt.Errorf("invalid reserved-ball status presentation")
	}
	for _, prompt := range []PlayerPrompt{preparation.Reserved, preparation.Relaunch} {
		if len(prompt.Text) != 20 || prompt.PlayerCell < 0 || prompt.PlayerCell >= 20 {
			return fmt.Errorf("invalid player status template")
		}
	}
	if a.Launch.SpringSensor == 0 || a.Launch.PlaySensor == 0 || a.Launch.SpringSensor == a.Launch.PlaySensor {
		return fmt.Errorf("invalid launch sensor identities")
	}
	if a.Tilt.TiltMeter < 1 || a.Tilt.TiltMeter > 65535 || a.Tilt.WarningMeter < 0 || a.Tilt.WarningMeter >= a.Tilt.TiltMeter || a.Tilt.DecayFrames < 1 || a.Tilt.DecayFrames > 65535 || a.Tilt.InitialDecay < 1 || a.Tilt.InitialDecay > 65535 || len(a.Tilt.Text) != 20 || a.Tilt.EntryCue.Action != "music" {
		return fmt.Errorf("invalid tilt presentation")
	}
	if a.Tilt.WarningMeter > 0 && a.Tilt.Warning.Action != "note" && a.Tilt.Warning.Action != "music" {
		return fmt.Errorf("missing tilt warning cue")
	}
	ui := a.ScoreEntry
	if err := a.Quit.Validate(); err != nil {
		return err
	}
	if len(ui.Prompt) > 20 || ui.PlayerCell < 0 || ui.PlayerCell >= len(ui.Prompt) || ui.InitialsCell < 0 || ui.InitialsCell+3 > len(ui.Prompt) || len(ui.Allowed) < 2 || ui.GameOverDelay < 1 || ui.PlayerDelay < 1 || ui.Repeats < 1 || ui.GameOverLoopBytes < 2 {
		return fmt.Errorf("invalid score-entry presentation")
	}
	check := func(p *IndexedImage, width, height int) error {
		if p.Width != width || p.Height != height || len(p.Pixels) != width*height || len(p.Colors) == 0 || len(p.Colors) > 256 {
			return fmt.Errorf("invalid native table image %dx%d", width, height)
		}
		for _, index := range p.Pixels {
			if int(index) >= len(p.Colors) {
				return fmt.Errorf("invalid native table palette index")
			}
		}
		return nil
	}
	for _, field := range []struct {
		image *IndexedImage
		w, h  int
	}{{&a.Art, 320, 516}, {&a.Ball, 16, 15}, {&a.Header, 320, 23}} {
		if err := check(field.image, field.w, field.h); err != nil {
			return err
		}
	}
	for _, f := range a.Flippers {
		for n := range f.Art {
			im := f.Art[n]
			if err := check(&im, 64, 43); err != nil {
				return err
			}
			if len(f.Mask[n]) != 64*43 {
				return fmt.Errorf("invalid native flipper silhouette")
			}
		}
	}
	for _, glyph := range a.Font {
		if glyph.Width == 0 {
			continue
		}
		if err := check(&glyph, 16, 19); err != nil {
			return err
		}
	}
	gauge := a.SpringGauge
	if gauge.UnitsPerLevel < 1 || len(gauge.Frames) != a.SpringLimit/gauge.UnitsPerLevel+1 || gauge.X < 0 || gauge.Y < 0 || gauge.X+11 > 320 || gauge.Y+18 > 516 {
		return fmt.Errorf("invalid native spring gauge")
	}
	for _, frame := range gauge.Frames {
		if err := check(&frame, 11, 18); err != nil {
			return err
		}
	}
	for _, layer := range a.Layers {
		if len(layer.Collision) == 0 || len(layer.Collision)%320 != 0 || layer.HiddenWidth != 336 || layer.HiddenHeight != 544 || len(layer.Hidden) != layer.HiddenWidth*layer.HiddenHeight || len(layer.GravityEdges) != len(layer.Collision)/320*2 {
			return fmt.Errorf("invalid native collision level")
		}
		for _, material := range layer.Collision {
			if material > 7 {
				return fmt.Errorf("invalid native collision material")
			}
		}
	}
	for _, id := range a.ProbeOrder {
		if id < 0 || id >= len(a.Points) {
			return fmt.Errorf("invalid native ball probe")
		}
	}
	for _, overlap := range a.CollisionOverlaps {
		if overlap.FromLayer < 0 || overlap.FromLayer >= len(a.Layers) || overlap.ToLayer < 0 || overlap.ToLayer >= len(a.Layers) || overlap.FromLayer == overlap.ToLayer || overlap.FromIndex < 0 || overlap.ToIndex < 0 || overlap.Length < 1 || overlap.FromIndex+overlap.Length > len(a.Layers[overlap.FromLayer].Collision) || overlap.ToIndex+overlap.Length > len(a.Layers[overlap.ToLayer].Collision) {
			return fmt.Errorf("invalid shared collision region")
		}
	}
	return nil
}
