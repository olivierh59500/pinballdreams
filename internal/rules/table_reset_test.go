package rules

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

func TestTableResetRestoresJackpotsAndKeepsLinkedAwards(t *testing.T) {
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
			for _, stage := range r.defs.Stages {
				if stage.Effect.Kind != ruledata.CollectJackpot {
					continue
				}
				bank := stage.Effect.Bank
				definition := r.defs.Jackpots[bank]
				r.jackpots[bank] += 2 * definition.Increment
				r.applyStageEffect(stage)
				r.jackpots[bank] += definition.Increment
				r.awardSources[bank], r.savedAwards[bank] = 7_654_321, 1_617_273
				r.Frame = 6254
				r.ResetTableAttract()
				if r.jackpots[bank] != definition.Initial || r.collectedJackpots[bank] != definition.Initial {
					t.Fatal("a new table cycle retained the previous running or collected jackpot")
				}
				if r.Frame != 6254 || r.awardSources[bank] != 7_654_321 || r.savedAwards[bank] != 1_617_273 {
					t.Fatal("table reset changed its shared clock or linked feature-award banks")
				}
				return
			}
			t.Fatal("supplied jackpot collection stage is missing")
		})
	}
}
