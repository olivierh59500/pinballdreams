package frontend

import (
	"image"
	"image/draw"

	"github.com/olivierh59500/pinballdreams/internal/resources"
)

type OpeningPhase string

const (
	Starting       OpeningPhase = "starting"
	NextPicture    OpeningPhase = "next-picture"
	PictureFadeIn  OpeningPhase = "picture-fade-in"
	PictureHold    OpeningPhase = "picture-hold"
	PictureFadeOut OpeningPhase = "picture-fade-out"
	CreditsFadeIn  OpeningPhase = "credits-fade-in"
	CreditsScroll  OpeningPhase = "credits-scroll"
	CreditsFadeOut OpeningPhase = "credits-fade-out"
	Closing        OpeningPhase = "closing"
	Exiting        OpeningPhase = "exiting"
	Finished       OpeningPhase = "finished"
)

// OpeningState exposes meaningful presentation state for offline inspection.
// Source-specific evidence is adapted by cmd/verify-intro, outside the game.
type OpeningState struct {
	Phase           OpeningPhase
	Picture         string
	Remaining       int
	Colors          [32]resources.Ink
	Glyphs          []PlacedGlyph
	ExtendedCredits bool
}

// OpeningInput contains host-independent actions for one presentation wait.
// ExtraCredits is a held secondary control, sampled as the developer logo fades.
type OpeningInput struct{ Skip, ExtraCredits bool }

// Intro presents a sequence of pictures followed by scrolling credits.
// The fixed presentation clock is shared by the desktop and Android hosts.
type Intro struct {
	assets       *resources.Opening
	phase        OpeningPhase
	scene        int
	delay        int
	colors       [32]resources.Ink
	black        [32]resources.Ink
	credits      *credits
	frame        *image.RGBA
	dirty        bool
	extraCredits bool
}

func NewIntro(assets *resources.Opening) *Intro {
	return &Intro{assets: assets, phase: Starting, scene: -1, frame: blackFrame(assets.Width, assets.Height), dirty: true}
}

func (i *Intro) Size() image.Point { return i.frame.Bounds().Size() }

// Tick advances one presentation frame. Skip fades the currently visible
// picture to black; it is available throughout the opening on both hosts.
func (i *Intro) Tick(input OpeningInput) {
	if i.credits != nil {
		i.credits.placements = i.credits.placements[:0]
	}
	if i.Done() {
		return
	}
	if input.Skip {
		i.phase = Exiting
	}
	if i.phase == Exiting {
		i.dirty = true
		if fade(i.colors[:], i.black[:]) {
			i.phase = Finished
			i.delay = 0
		}
		return
	}
	if i.delay > 0 {
		i.delay--
		return
	}
	i.dirty = true
	switch i.phase {
	case Starting:
		i.delay = i.assets.StartupFrames
		i.phase = NextPicture
	case NextPicture:
		i.showNextPicture()
	case PictureFadeIn:
		scene := &i.assets.Scenes[i.scene]
		if fade(i.colors[:], scene.Picture.Colors) {
			i.delay = scene.HoldFrames
			i.phase = PictureFadeOut
		}
	case PictureFadeOut:
		if i.assets.Scenes[i.scene].Picture.Name == "developer-logo" {
			i.extraCredits = input.ExtraCredits
		}
		if fade(i.colors[:], i.black[:]) {
			i.phase = NextPicture
		}
	case CreditsFadeIn:
		i.credits.tick()
		if fade(i.colors[:], i.assets.Credits.Colors) {
			i.phase = CreditsScroll
		}
	case CreditsScroll:
		i.credits.tick()
		if i.credits.finished {
			i.phase = CreditsFadeOut
		}
	case CreditsFadeOut:
		i.credits.tick()
		if fade(i.colors[:], i.black[:]) {
			i.phase = Closing
		}
	case Closing:
		i.phase = Exiting
	}
}

func (i *Intro) showNextPicture() {
	i.scene++
	clear(i.colors[:])
	if i.scene == len(i.assets.Scenes) {
		i.credits = newCredits(&i.assets.Credits, i.extraCredits)
		i.phase = CreditsFadeIn
		return
	}
	if i.assets.Scenes[i.scene].StartWhite {
		for n := range i.colors {
			i.colors[n] = resources.Ink{R: 15, G: 15, B: 15}
		}
	}
	i.phase = PictureFadeIn
}

func (i *Intro) Done() bool { return i.phase == Finished }

func (i *Intro) State() OpeningState {
	state := OpeningState{Phase: i.phase, Remaining: i.delay, Colors: i.colors}
	if i.scene >= 0 && i.scene < len(i.assets.Scenes) {
		state.Picture = i.assets.Scenes[i.scene].Picture.Name
	}
	if i.phase == PictureFadeOut && i.delay > 0 {
		state.Phase = PictureHold
	}
	if i.credits != nil {
		state.ExtendedCredits = i.extraCredits
		state.Picture = "credits"
		state.Glyphs = append([]PlacedGlyph(nil), i.credits.placements...)
	}
	return state
}

// Frame returns a reused, opaque canvas. Low-resolution pictures scale by
// whole pixels; credits retain their original 640-pixel horizontal detail.
func (i *Intro) Frame() *image.RGBA {
	if !i.dirty {
		return i.frame
	}
	if i.credits != nil {
		picture := resources.Picture{Colors: i.assets.Credits.Colors}
		palette := picture.Palette(i.colors[:2])
		draw.Draw(i.frame, i.frame.Bounds(), image.NewUniform(palette[0]), image.Point{}, draw.Src)
		for y := range creditHeight {
			row := i.frame.Pix[y*2*i.frame.Stride:]
			paintRow(row, i.credits.row(y), palette, 1)
			copy(row[i.frame.Stride:i.frame.Stride*2], row[:i.frame.Stride])
		}
	} else if i.scene >= 0 {
		scene := &i.assets.Scenes[i.scene]
		palette := scene.Picture.Palette(i.colors[:len(scene.Picture.Colors)])
		draw.Draw(i.frame, i.frame.Bounds(), image.NewUniform(palette[0]), image.Point{}, draw.Src)
		paintPicture(i.frame, &scene.Picture, i.colors[:], scene.Y, 2)
	}
	i.dirty = false
	return i.frame
}
