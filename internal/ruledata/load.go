package ruledata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
)

func Filename(table int) string { return fmt.Sprintf("table%d-rules.json", table+1) }

// LoadFS accepts a native rules bundle only with its matching source record.
func LoadFS(source fs.FS, table int, record []byte) (*Definitions, error) {
	return LoadNativeFS(source, table, fmt.Sprintf("%x", sha256.Sum256(record)))
}

func LoadNativeFS(source fs.FS, table int, expectedSHA256 string) (*Definitions, error) {
	data, err := fs.ReadFile(source, Filename(table))
	if err != nil {
		return nil, fmt.Errorf("extract native table rules first: %w", err)
	}
	var d Definitions
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("native table rules: %w", err)
	}
	if d.TableID != table || d.SourceSHA256 != expectedSHA256 {
		return nil, fmt.Errorf("native rules do not match table %d; extract the supplied disks again", table+1)
	}
	return &d, d.Validate()
}

// Validate checks the schema and relationships needed by the native engine.
func (d *Definitions) Validate() error {
	if d.Version != 13 || d.TableID < 0 || d.TableID > 3 || len(d.SourceSHA256) != 64 || len(d.Stages) == 0 || len(d.Groups) == 0 || len(d.Lamps) == 0 || len(d.Layers) != 2 || d.Paths == nil || d.Objects == nil || d.Features == nil || d.Events == nil || d.Sounds == nil || d.Announcements == nil || d.Texts == nil || d.LampDrawBudget < 1 {
		return fmt.Errorf("invalid native rules schema")
	}
	if d.Relaunch.Mode != RelaunchSingle && d.Relaunch.Mode != RelaunchCounted || d.Relaunch.Indicator != d.ExtraBallLamp {
		return fmt.Errorf("invalid native relaunch policy")
	}
	if d.Relaunch.Mode == RelaunchCounted {
		if _, ok := d.Lamps[d.Relaunch.Indicator]; !ok {
			return fmt.Errorf("missing relaunch indicator")
		}
	}
	if d.Panel.InitialDelay < 1 || d.Panel.RepeatDelay < 1 || len(d.Panel.Idle) == 0 {
		return fmt.Errorf("invalid native live panel schedule")
	}
	for _, id := range d.Panel.Idle {
		if _, ok := d.Announcements[id]; !ok {
			return fmt.Errorf("missing idle panel announcement")
		}
	}
	for _, binding := range d.Panel.Bindings {
		text, ok := d.Texts[binding.Text]
		if !ok || binding.Position < 0 || binding.Digits < 1 || binding.Digits > 10 || binding.Position+binding.Digits >= len(text) {
			return fmt.Errorf("invalid native panel template field")
		}
		switch binding.Kind {
		case "player", "ball", "multiplier", "advance-tens":
		case "jackpot", "collected-jackpot", "award-source", "saved-award":
			if _, ok := d.Jackpots[binding.Bank]; !ok {
				return fmt.Errorf("unknown panel award bank")
			}
		default:
			return fmt.Errorf("unknown native panel field")
		}
	}
	if special := d.Panel.Special; special.Stage != 0 {
		if _, ok := d.Stages[special.Stage]; !ok {
			return fmt.Errorf("unknown saved-value panel stage")
		}
		if _, ok := d.Jackpots[special.Bank]; !ok || special.Priority < 1 || special.ClockMask < 1 || special.Delta == 0 || special.Delta > 1_000_000_000_000 || len(d.Texts[special.Text]) == 0 {
			return fmt.Errorf("invalid saved-value panel parameters")
		}
	}
	if len(d.SkillLamps) > 0 && (d.SkillGroup == 0) == (d.SkillPath == 0) {
		return fmt.Errorf("invalid native skill reward")
	}
	for _, id := range d.SkillLamps {
		if _, ok := d.Lamps[id]; !ok {
			return fmt.Errorf("missing native skill lamp %d", id)
		}
	}
	if d.SkillPath != 0 {
		if _, ok := d.Paths[d.SkillPath]; !ok {
			return fmt.Errorf("missing native skill path %d", d.SkillPath)
		}
	}
	if d.BallStartPath != 0 {
		if _, ok := d.Paths[d.BallStartPath]; !ok {
			return fmt.Errorf("missing native new-ball path %d", d.BallStartPath)
		}
	}
	cycle := d.TargetCycle
	if d.Bonus.EntryFrames < 1 || len(d.Bonus.EntryText) != 20 || d.Bonus.MultiplierDelay < 1 {
		return fmt.Errorf("invalid native end-ball presentation")
	}
	for _, id := range d.Bonus.EntrySounds {
		if _, ok := d.Sounds[id]; !ok {
			return fmt.Errorf("missing native end-ball entry cue")
		}
	}
	if _, ok := d.Sounds[d.Bonus.FinalHoldSound]; !ok {
		return fmt.Errorf("missing native final held-bonus cue")
	}
	if _, ok := d.Announcements[d.Bonus.FinalHoldMessage]; !ok {
		return fmt.Errorf("missing native final held-bonus announcement")
	}
	if cycle.ResetDelay < 1 || cycle.ResetPeriod < 1 || cycle.RetriggerFrames < 1 {
		return fmt.Errorf("invalid target timing")
	}
	if _, ok := d.Sounds[cycle.ResetSound]; !ok {
		return fmt.Errorf("missing target reset sound")
	}
	if _, ok := d.Sounds[d.CaptureReleaseSound]; !ok {
		return fmt.Errorf("missing capture release sound")
	}
	for id, stage := range d.Stages {
		if stage.ID != id || stage.Kind != Collectible && stage.Kind != Repeatable {
			return fmt.Errorf("invalid native stage %d", id)
		}
		if copy := stage.Effect.CopyText; copy != nil {
			if copy.Characters < 1 || copy.Characters > 20 || len(d.Texts[copy.Source]) <= copy.Characters || len(d.Texts[copy.Destination]) <= copy.Characters {
				return fmt.Errorf("invalid native award text copy")
			}
		}
		switch stage.Effect.Kind {
		case NoEffect, SwitchStages, AnnounceAdvance, CopyLeadingScore, ExtraBall, SetMultiplier, HoldBonus, ProgressFeature, RaiseJackpot, CollectJackpot, FixedAward, SavedAward, CopyScore, CopyBonus, ArmLocks, ReserveBall, CollectLockedJackpot, ChangeGate, CountBonus, SelectRoulette, Untranslated:
		default:
			return fmt.Errorf("invalid native stage effect %q", stage.Effect.Kind)
		}
	}
	for _, group := range d.Groups {
		for _, id := range group.Lamps {
			if _, ok := d.Lamps[id]; !ok {
				return fmt.Errorf("missing native lamp %d", id)
			}
		}
	}
	for id, path := range d.Paths {
		if path.Mode != Sequential && path.Mode != Selected && path.Mode != Together {
			return fmt.Errorf("invalid feature path %d", id)
		}
		for _, step := range path.Steps {
			if _, ok := d.Stages[step.Stage]; !step.Stop && step.Path == 0 && !ok {
				return fmt.Errorf("missing feature-path stage %d", step.Stage)
			}
		}
	}
	for id, message := range d.Announcements {
		if message.ID != id || len(d.Texts[message.Text]) == 0 {
			return fmt.Errorf("invalid native announcement %d", id)
		}
	}
	for _, slots := range d.TextSlots {
		for _, slot := range slots {
			if slot.Offset < 0 || slot.Offset >= len(d.Texts[slot.Text])-1 {
				return fmt.Errorf("invalid native text slot")
			}
		}
	}
	return nil
}
