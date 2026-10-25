package frontend

import (
	"image"

	"github.com/olivierh59500/pinballdreams/internal/resources"
)

// Menu owns a native palette fade and the four extracted selection regions.
type Menu struct {
	assets         *resources.TableSelection
	colors         [][]resources.Ink
	black          [][]resources.Ink
	frame          *image.RGBA
	ready          bool
	dirty          bool
	hiding, hidden bool
}

func NewMenu(assets *resources.TableSelection) *Menu {
	m := &Menu{assets: assets, frame: blackFrame(320, 256), dirty: true}
	for _, band := range assets.Bands {
		m.colors = append(m.colors, make([]resources.Ink, len(band.Picture.Colors)))
		m.black = append(m.black, make([]resources.Ink, len(band.Picture.Colors)))
	}
	return m
}

func (m *Menu) Tick() {
	if m.hidden || m.ready && !m.hiding {
		return
	}
	m.ready = true
	for n, band := range m.assets.Bands {
		target := band.Picture.Colors
		if m.hiding {
			target = m.black[n]
		}
		if !fade(m.colors[n], target) {
			m.ready = false
		}
	}
	if m.ready && m.hiding {
		m.hidden = true
		m.ready = false
	}
	m.dirty = true
}

func (m *Menu) Ready() bool  { return m.ready }
func (m *Menu) Hide()        { m.hiding = true; m.ready = false }
func (m *Menu) Hidden() bool { return m.hidden }

func (m *Menu) TableAt(x, y int) int {
	for n, region := range m.assets.Regions {
		if image.Pt(x, y).In(region) {
			return n + 1
		}
	}
	return 0
}

func (m *Menu) Frame() *image.RGBA {
	if m.dirty {
		for n, band := range m.assets.Bands {
			paintPicture(m.frame, &band.Picture, m.colors[n], band.Y, 1)
		}
		m.dirty = false
	}
	return m.frame
}
