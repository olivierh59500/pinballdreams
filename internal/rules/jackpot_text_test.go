package rules

import (
	"strings"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

func TestCollectedJackpotLabelSurvivesRunningBankReset(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			table, err := resources.LoadFS(assets.Files(), id)
			if err != nil {
				t.Fatal(err)
			}
			r, err := New(table)
			if err != nil {
				t.Fatal(err)
			}
			var collection Stage
			for _, stage := range r.defs.Stages {
				if stage.Effect.Kind == ruledata.CollectJackpot {
					collection = stage
					break
				}
			}
			bank := collection.Effect.Bank
			r.jackpots[bank] = 5_455_668
			r.UpdatePanelTemplates(2)
			// An earlier action can raise the bank after its last status write.
			// Ignition snapshots that published label; later tables display
			// the amount actually collected before restoring the running bank.
			r.jackpots[bank] = 5_569_585
			var awarded uint64
			r.OnAward = func(points, _ uint64) { awarded += points }
			r.applyStageEffect(collection)
			if awarded != 5_569_585 || r.jackpots[bank] != r.defs.Jackpots[bank].Initial {
				t.Fatal("collecting changed the award or failed to restore the running bank")
			}
			r.NewBall(0)
			r.UpdatePanelTemplates(3)
			want := "5569585"
			if collection.Effect.CopyText != nil {
				want = "5455668"
			}
			message := r.defs.Announcements[collection.OnMessage]
			if text := string(r.messageText(message.Text)); !strings.Contains(text, want) {
				t.Fatalf("collected label %q does not retain %s", text, want)
			}
		})
	}
}

func TestLockedJackpotLabelKeepsBaseAmountAndCopies(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 3)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	var collection Stage
	for _, stage := range r.defs.Stages {
		if stage.Effect.Kind == ruledata.CollectLockedJackpot {
			collection = stage
		}
	}
	r.jackpots[r.defs.Locks.SuperBank] = 5_455_668
	for i := range r.locks.Slots {
		r.locks.Slots[i] = true
	}
	var awarded uint64
	r.OnAward = func(points, _ uint64) { awarded += points }
	r.applyStageEffect(collection)
	r.UpdatePanelTemplates(2)
	message := r.defs.Announcements[collection.OnMessage]
	text := string(r.messageText(message.Text))
	if awarded != 3*5_455_668 || !strings.Contains(text, "3 X") || !strings.Contains(text, "5455668") {
		t.Fatalf("locked award %d and label %q lost their base amount or multiplier", awarded, text)
	}
}
