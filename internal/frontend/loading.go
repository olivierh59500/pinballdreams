package frontend

import (
	"image"

	"github.com/olivierh59500/pinballdreams/internal/resources"
)

// Loading fades the extracted logo and selected title on one shared palette.
// Its image and position are plain assets, independent of source display code.
type Loading struct {
	picture      *resources.Picture
	y            int
	colors       []resources.Ink
	frame        *image.RGBA
	ready, dirty bool
}

func NewLoading(assets *resources.LoadingScreen, table int) *Loading {
	picture := &assets.Pictures[table]
	return &Loading{picture: picture, y: assets.Y, colors: make([]resources.Ink, len(picture.Colors)), frame: blackFrame(320, 256), dirty: true}
}

func (l *Loading) Tick() {
	if l.ready {
		return
	}
	l.ready = fade(l.colors, l.picture.Colors)
	l.dirty = true
}

func (l *Loading) Ready() bool { return l.ready }

func (l *Loading) Frame() *image.RGBA {
	if l.dirty {
		paintPicture(l.frame, l.picture, l.colors, l.y, 1)
		l.dirty = false
	}
	return l.frame
}
