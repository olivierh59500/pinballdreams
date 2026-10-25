package rules

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestArtworkFramesStayStableUntilPresentationAdvances(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	first := r.Art(false, false)
	past := append([]byte(nil), first.Pix...)
	if r.Art(false, false) != first {
		t.Fatal("identical artwork state did not reuse its composition")
	}
	for _, group := range r.defs.Groups {
		changed := false
		for _, id := range group.Lamps {
			if len(r.defs.Lamps[id].Pixels) == 0 {
				continue
			}
			on := r.lampVisible(id, false, false)
			r.Lamps[id].Flags, r.Lamps[id].Players = 0, 0
			if !on {
				r.Lamps[id].Players = r.mask()
			}
			changed = true
			break
		}
		if changed {
			break
		}
	}
	if r.Art(false, false) != first {
		t.Fatal("a read advanced pending lamp changes")
	}
	r.advanceArtwork(false, false)
	second := r.Art(false, false)
	if second == first || !bytes.Equal(first.Pix, past) {
		t.Fatal("a new lamp state reused or changed a previously published frame")
	}
	if r.Art(false, false) != second {
		t.Fatal("the changed artwork state did not become reusable")
	}
	r.advanceArtwork(false, true)
	if r.Art(false, true) == second {
		t.Fatal("tilt did not invalidate the artwork composition")
	}
}

func TestBorrowedArtworkReusesStorageWithoutInvalidatingSnapshots(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	view := r.ArtView(false, false)
	lampID := 0
	for _, group := range r.defs.Groups {
		for _, id := range group.Lamps {
			if len(r.defs.Lamps[id].Pixels) > 0 {
				lampID = id
				break
			}
		}
		if lampID != 0 {
			break
		}
	}
	if lampID == 0 {
		t.Fatal("no supplied lamp mask for ownership check")
	}
	toggle := func() {
		on := r.lampVisible(lampID, false, false)
		lamp := r.Lamps[lampID]
		lamp.Flags = 0
		lamp.Players = 0
		if !on {
			lamp.Players = r.mask()
		}
		r.advanceArtwork(false, false)
	}
	toggle()
	if r.ArtView(false, false) != view {
		t.Fatal("an unretained borrowed image caused a full-surface copy")
	}
	snapshot := r.Art(false, false)
	pixels := append([]byte(nil), snapshot.Pix...)
	if r.ArtView(false, false) != snapshot {
		t.Fatal("borrowing changed the current surface")
	}
	toggle()
	if r.ArtView(false, false) == snapshot || !bytes.Equal(snapshot.Pix, pixels) {
		t.Fatal("a borrowed read invalidated the earlier stable snapshot")
	}
}
