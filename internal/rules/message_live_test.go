package rules

import (
	"strings"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestCountingAnnouncementReadsUpdatedAwardDuringReveal(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 2)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	special := r.defs.Panel.Special
	r.savedAwards[special.Bank] = 1_604_736
	r.UpdatePanelTemplates(2)
	r.announce(r.defs.Stages[special.Stage].OnMessage, 100)
	for range 5 {
		r.advanceAnnouncement()
	}
	// The original continuous mission updates its saved value once more
	// before capture. Its mask animation must reveal that collected amount.
	r.savedAwards[special.Bank] = 1_617_273
	r.UpdatePanelTemplates(2)
	for range 80 {
		r.advanceAnnouncement()
		if strings.Contains(r.message, "161727") {
			return
		}
	}
	t.Fatal("counting announcement retained the amount from before collection")
}

func TestAttractAnimationKeepsItsSuppliedText(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	animation := r.NewAttractAnimation(table.Attract.Roaming[0])
	for range 2 {
		text, wrote, _ := animation.Step()
		if wrote && strings.TrimSpace(text) == "-- PINBALL DREAMS --" {
			return
		}
	}
	t.Fatal("attract animation lost its supplied title outside the live templates")
}
