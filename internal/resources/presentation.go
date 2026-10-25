package resources

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io/fs"

	"github.com/olivierh59500/pinballdreams/internal/highscore"
)

const PresentationFile = "presentation.json"

// Ink retains the sixteen intensity levels in the supplied artwork. Keeping
// that precision makes fades deterministic without introducing new colors.
type Ink struct{ R, G, B uint8 }

// Picture contains ordinary indexed pixels. Shaded pictures additionally use
// palette entries 32..63 at half the intensity of entries 0..31.
type Picture struct {
	Name          string
	Width, Height int
	Pixels        []byte
	Colors        []Ink
	Shaded        bool
}

// Palette expands the current colors once per frame, outside the pixel loop.
func (p *Picture) Palette(colors []Ink) []color.RGBA {
	count := len(colors)
	if p.Shaded {
		count *= 2
	}
	result := make([]color.RGBA, count)
	for n, c := range colors {
		result[n] = color.RGBA{c.R * 17, c.G * 17, c.B * 17, 255}
		if p.Shaded {
			result[n+len(colors)] = color.RGBA{c.R / 2 * 17, c.G / 2 * 17, c.B / 2 * 17, 255}
		}
	}
	return result
}

// OpeningScene describes one picture's placement and presentation timing.
type OpeningScene struct {
	Picture    Picture
	Y          int
	HoldFrames int
	StartWhite bool
}

type Alignment string

const (
	Centered Alignment = "center"
	Left     Alignment = "left"
	Right    Alignment = "right"
)

// CreditLine is fully decoded text. Runtime layout never reads source escapes.
type CreditLine struct {
	Text        string
	Align       Alignment
	DelayFrames int
}

// Glyph uses a tight alpha mask and a separate advance for proportional text.
type Glyph struct {
	Advance       int
	Width, Height int
	Mask          []byte
}

type Credits struct {
	Font         [96]Glyph
	Lines        []CreditLine
	ExtraLines   []CreditLine
	Colors       []Ink
	InitialDelay int
	ScrollRows   int
}

type Opening struct {
	Width, Height int
	StartupFrames int
	Scenes        []OpeningScene
	Credits       Credits
}

type MenuBand struct {
	Picture Picture
	Y       int
}

type TableSelection struct {
	Bands      []MenuBand
	Regions    [4]image.Rectangle
	HoldFrames int
	ExitFrames int
}

// SelectionFooter contains the small source font and the two independent text
// streams shared by selection and rankings. Runtime text uses ASCII directly.
type SelectionFooter struct {
	Font                        [128]Glyph
	Ticker                      string
	Shortcuts                   []TickerShortcut
	Captions                    []string
	TickerColors, CaptionColors []Ink
	TickerY, CaptionY           int
	CharacterFrames, HoldFrames int
}

// TickerShortcut selects one temporary message using ordinary letter keys.
// The normal ticker resumes when the selected message reaches its end.
type TickerShortcut struct{ Keys, Text string }

type ScoreGlyph struct {
	Width, Height int
	Pixels        []byte
}

// Scoreboard contains supplied art, glyphs, rankings and their music cadence.
type Scoreboard struct {
	Background          Picture
	Font                [256]ScoreGlyph
	InitialGlyphs       [128]byte
	Positions           [16]image.Point
	Defaults            highscore.Book
	HoldFrames          int
	MusicRetriggerPhase [16]byte
}

// Presentation is produced by the Go extractor. The runtime consumes this
// named, validated data rather than locating assets inside an executable.
type Presentation struct {
	Version        int
	InitialLowPass bool
	Opening        Opening
	Menu           TableSelection
	Scores         Scoreboard
	Footer         SelectionFooter
	Loading        LoadingScreen
}

// LoadingScreen contains the source loading logo and selected table title,
// already combined into ordinary indexed pictures at their declared position.
type LoadingScreen struct {
	Pictures [4]Picture
	Y        int
}

func PresentationFS(source fs.FS) (*Presentation, error) {
	data, err := fs.ReadFile(source, PresentationFile)
	if err != nil {
		return nil, fmt.Errorf("extract the supplied ADFs first: %w", err)
	}
	var p Presentation
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("presentation assets: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate rejects corrupt generated files before animation or rendering.
func (p *Presentation) Validate() error {
	if p.Version != 9 || p.Opening.Width != 672 || p.Opening.Height != 512 || p.Opening.Credits.ScrollRows != 281 || len(p.Opening.Scenes) != 4 || len(p.Menu.Bands) != 4 || len(p.Opening.Credits.Lines) == 0 || len(p.Opening.Credits.ExtraLines) != 21 {
		return fmt.Errorf("unsupported presentation asset layout")
	}
	picture := func(im *Picture) error {
		if im.Width != 320 || im.Height < 1 || im.Height > 256 || len(im.Pixels) != im.Width*im.Height || len(im.Colors) == 0 || len(im.Colors) > 32 || im.Shaded && len(im.Colors) != 32 {
			return fmt.Errorf("invalid presentation picture %q", im.Name)
		}
		count := len(im.Colors)
		if im.Shaded {
			count *= 2
		}
		for _, index := range im.Pixels {
			if int(index) >= count {
				return fmt.Errorf("invalid pixel in presentation picture %q", im.Name)
			}
		}
		return validInks(im.Colors)
	}
	for _, im := range p.Loading.Pictures {
		if err := picture(&im); err != nil {
			return err
		}
		if im.Height != 17 || len(im.Colors) != 8 || p.Loading.Y < 0 || p.Loading.Y+im.Height > 256 {
			return fmt.Errorf("invalid loading picture")
		}
	}
	for _, scene := range p.Opening.Scenes {
		if err := picture(&scene.Picture); err != nil {
			return err
		}
		if scene.Y < 0 || scene.Y+scene.Picture.Height > 256 || scene.HoldFrames < 0 {
			return fmt.Errorf("invalid opening scene %q", scene.Picture.Name)
		}
	}
	for _, band := range p.Menu.Bands {
		if err := picture(&band.Picture); err != nil {
			return err
		}
		if band.Y < 0 || band.Y+band.Picture.Height > 256 {
			return fmt.Errorf("invalid selection band %q", band.Picture.Name)
		}
	}
	footer := &p.Footer
	if footer.TickerY < 0 || footer.TickerY+8 > footer.CaptionY || footer.CaptionY+8 > 256 || footer.CharacterFrames != 8 || footer.HoldFrames < 1 || len(footer.Ticker) == 0 || len(footer.Captions) == 0 || len(footer.TickerColors) != 4 || len(footer.CaptionColors) != 4 {
		return fmt.Errorf("invalid selection footer")
	}
	if err := validInks(footer.TickerColors); err != nil {
		return err
	}
	if err := validInks(footer.CaptionColors); err != nil {
		return err
	}
	for _, glyph := range footer.Font {
		if glyph.Width != 8 || glyph.Height != 8 || glyph.Advance != 8 || len(glyph.Mask) != 64 {
			return fmt.Errorf("invalid selection footer glyph")
		}
		for _, pixel := range glyph.Mask {
			if pixel > 1 {
				return fmt.Errorf("invalid selection footer glyph pixel")
			}
		}
	}
	texts := append([]string{footer.Ticker}, footer.Captions...)
	if len(footer.Shortcuts) != 6 {
		return fmt.Errorf("incomplete selection ticker shortcuts")
	}
	seen := make(map[string]bool)
	for _, shortcut := range footer.Shortcuts {
		if len(shortcut.Keys) < 2 || len(shortcut.Keys) > 32 || shortcut.Text == "" || seen[shortcut.Keys] {
			return fmt.Errorf("invalid selection ticker shortcut")
		}
		seen[shortcut.Keys] = true
		for _, key := range []byte(shortcut.Keys) {
			if key < 'A' || key > 'Z' {
				return fmt.Errorf("invalid selection ticker shortcut key")
			}
		}
		texts = append(texts, shortcut.Text)
	}
	for _, text := range texts {
		for _, c := range []byte(text) {
			if c < 32 || c >= 128 {
				return fmt.Errorf("invalid selection footer character")
			}
		}
	}
	for _, caption := range footer.Captions {
		if len(caption) != 40 {
			return fmt.Errorf("invalid selection footer caption")
		}
	}
	if err := picture(&p.Scores.Background); err != nil {
		return err
	}
	if p.Menu.HoldFrames < 1 || p.Menu.ExitFrames != 16 || p.Scores.HoldFrames < 1 {
		return fmt.Errorf("invalid attract timing")
	}
	if err := p.Scores.Defaults.Validate(" ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"); err != nil {
		return err
	}
	for _, point := range p.Scores.Positions {
		if point.X < 0 || point.X+112 > 320 || point.Y < 0 || point.Y+10 > 256 {
			return fmt.Errorf("invalid high-score placement")
		}
	}
	hasMusicCadence := false
	for interval, phase := range p.Scores.MusicRetriggerPhase {
		if interval == 0 && phase != 0 || interval > 0 && int(phase) >= interval {
			return fmt.Errorf("invalid ranking music repeat cadence")
		}
		hasMusicCadence = hasMusicCadence || phase != 0
	}
	if !hasMusicCadence {
		return fmt.Errorf("missing ranking music repeat cadence")
	}
	for _, glyph := range p.Scores.Font {
		if glyph.Width != 8 || glyph.Height != 10 || len(glyph.Pixels) != 80 {
			return fmt.Errorf("invalid high-score glyph")
		}
		for _, index := range glyph.Pixels {
			if int(index) >= len(p.Scores.Background.Colors) {
				return fmt.Errorf("invalid high-score glyph color")
			}
		}
	}
	for _, glyph := range p.Opening.Credits.Font {
		if glyph.Advance < 0 || glyph.Width != 32 || glyph.Height != 23 || len(glyph.Mask) != glyph.Width*glyph.Height {
			return fmt.Errorf("invalid credit glyph")
		}
		for _, pixel := range glyph.Mask {
			if pixel > 1 {
				return fmt.Errorf("invalid credit glyph pixel")
			}
		}
	}
	for _, region := range p.Menu.Regions {
		if region.Empty() || !region.In(image.Rect(0, 0, 320, 256)) {
			return fmt.Errorf("invalid table selection region")
		}
	}
	if p.Opening.StartupFrames < 0 || p.Opening.Credits.InitialDelay < 0 || len(p.Opening.Credits.Colors) != 2 {
		return fmt.Errorf("invalid opening timing or colors")
	}
	if err := validInks(p.Opening.Credits.Colors); err != nil {
		return err
	}
	lines := append(append([]CreditLine(nil), p.Opening.Credits.Lines...), p.Opening.Credits.ExtraLines...)
	for _, line := range lines {
		if line.DelayFrames < 1 || line.Align != Centered && line.Align != Left && line.Align != Right {
			return fmt.Errorf("invalid credit line")
		}
		for _, c := range []byte(line.Text) {
			if c < 32 || c >= 128 {
				return fmt.Errorf("invalid credit character")
			}
		}
	}
	return nil
}

func validInks(colors []Ink) error {
	for _, c := range colors {
		if c.R > 15 || c.G > 15 || c.B > 15 {
			return fmt.Errorf("invalid presentation color")
		}
	}
	return nil
}
