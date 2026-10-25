package rules

import (
	"image"
)

// artworkState owns the presented table and the last drawn lamp states.
// Updating only changed lamps preserves ordering where masks overlap.
type artworkState struct {
	frame     *image.Paletted
	lampIDs   []int
	shown     []bool
	byPlayer  [8][]bool
	owner     int
	published bool
}

func (r *Engine) initializeArtwork() {
	source := r.defs.Table.Art
	a := &artworkState{frame: image.NewPaletted(source.Bounds(), source.Palette)}
	copy(a.frame.Pix, source.Pix)
	r.artwork = a
	for _, id := range r.defs.Gates {
		r.paintTarget(id)
	}
	for _, group := range r.defs.TargetGroups {
		for _, id := range group.Targets {
			r.paintTarget(id)
		}
	}
	for _, group := range r.defs.Groups {
		for _, id := range group.Lamps {
			a.lampIDs = append(a.lampIDs, id)
			a.shown = append(a.shown, false)
			a.paint(r.defs.Lamps[id].Pixels, false)
		}
	}
	for player := range a.byPlayer {
		a.byPlayer[player] = make([]bool, len(a.lampIDs))
		for index := range a.byPlayer[player] {
			a.byPlayer[player][index] = true
		}
	}
	a.byPlayer[r.actorIndex()] = a.shown
	a.owner = r.actorIndex()
}

// mutable preserves previously published frames while reusing a working image
// for successive changes within one presentation update.
func (a *artworkState) mutable() {
	if !a.published {
		return
	}
	old := a.frame
	a.frame = image.NewPaletted(old.Bounds(), old.Palette)
	copy(a.frame.Pix, old.Pix)
	a.published = false
}

func (a *artworkState) paint(pixels []image.Point, bright bool) {
	if len(pixels) == 0 {
		return
	}
	a.mutable()
	for _, point := range pixels {
		if !point.In(a.frame.Bounds()) {
			continue
		}
		at := point.Y*a.frame.Stride + point.X
		if bright {
			a.frame.Pix[at] &^= 32
		} else {
			a.frame.Pix[at] |= 32
		}
	}
}

func (r *Engine) paintTarget(id int) {
	if r.artwork != nil {
		r.artwork.paint(r.defs.Targets[id].Pixels, r.collected[id]&r.mask() == 0)
	}
}

func (r *Engine) lampVisible(id int, attract, tilted bool) bool {
	lamp := r.Lamps[id]
	if tilted {
		return false
	}
	if lamp.Flags&lampFlash != 0 {
		return lamp.Phase
	}
	on := lamp.Players&r.mask() != 0
	if lamp.Flags&lampBlinkWhenCollected != 0 {
		if !on {
			return false
		}
		on = false
	}
	if attract || lamp.Flags&lampAnimateDuringPlay != 0 {
		on = on || lamp.Blink
	}
	return on
}

// advanceArtwork applies the source's bounded changed-lamp service. Maskless
// records still consume a change slot because they own a cached draw state.
func (r *Engine) advanceArtwork(attract, tilted bool) {
	if r.artwork == nil {
		r.initializeArtwork()
	}
	a := r.artwork
	if a.owner != r.actorIndex() {
		a.shown = a.byPlayer[r.actorIndex()]
		a.owner = r.actorIndex()
	}
	remaining := r.defs.LampDrawBudget
	for index, id := range a.lampIDs {
		on := r.lampVisible(id, attract, tilted)
		if a.shown[index] == on {
			continue
		}
		a.shown[index] = on
		a.paint(r.defs.Lamps[id].Pixels, on)
		remaining--
		if remaining == 0 {
			break
		}
	}
}

// Art publishes the native table surface without advancing its drawing clock.
// A first standalone capture settles the initial view; live Tick calls advance
// presentation before collectors update their next-frame flashes.
func (r *Engine) Art(attract, tilted bool) *image.Paletted {
	view := r.ArtView(attract, tilted)
	r.artwork.published = true
	return view
}

// ArtView borrows the current indexed surface for immediate read-only work.
// Callers must finish reading before the next rule update and must not retain
// or modify the image. Art provides a stable published snapshot when needed.
func (r *Engine) ArtView(attract, tilted bool) *image.Paletted {
	if r.artwork == nil {
		r.initializeArtwork()
		for n := 0; n <= len(r.artwork.lampIDs)/r.defs.LampDrawBudget; n++ {
			r.advanceArtwork(attract, tilted)
		}
	}
	return r.artwork.frame
}
