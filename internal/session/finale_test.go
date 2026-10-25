package session

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestFinalHeldBonusCompletesBeforeScoreEntry(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(2)
			waitForPreparedBall(t, s)
			s.Player, s.bankPlayer, s.Rules.Player, s.Ball = 1, 1, 1, 3
			s.Players[1] = Player{Score: 900_000, Bonus: 12_340}
			s.Rules.BonusMultiplier = 3
			s.Rules.HoldBonus = true
			clock, display := s.Rules.Frame, s.panel.text()
			s.finishBall()
			if s.Player != 2 || s.Rules.Player != 2 || s.Ball != 4 || s.bankPlayer != 1 {
				t.Fatal("final selection was postponed until after the retained-bonus encore")
			}
			if s.finale == nil || !s.Playing || s.NameEntry != nil || s.Rules.Frame != clock || s.panel.text() != display {
				t.Fatal("last-ball selection did not enter the encore at the current wait boundary")
			}
			for n := 0; s.finale != nil && n < 2000; n++ {
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if s.finale != nil || !s.pendingGameOver || s.NameEntry != nil || s.Players[1].Score != 937_020 || s.Players[1].Bonus != 12_340 || s.Players[0].Score != 0 {
				t.Fatalf("final awards did not reach the outgoing player's bank before game over: %+v", s.Players)
			}
			if err := s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
			if s.Playing || s.pendingGameOver {
				t.Fatal("the following update did not enter game over")
			}
		})
	}
}

func TestFinalRelaunchCreditDefersHeldBonusEncore(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	waitForPreparedBall(t, s)
	s.Ball = 3
	s.Rules.HoldBonus = true
	s.Rules.RelaunchCredits = 1
	s.finishBall()
	if s.finale != nil || !s.pendingNewBall || !s.Playing || s.Ball != 3 || s.Player != 0 {
		t.Fatal("an unconsumed relaunch credit entered the last-ball finale")
	}
}

func TestTiltedLastBallKeepsTheHeldBonusBank(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(1)
			waitForPreparedBall(t, s)
			s.Ball = 3
			s.Players[0].Bonus = 12_340
			s.Rules.HoldBonus = true
			s.Engine.Tilted, s.pendingDrain = true, true
			s.enterTilt()
			clock := s.Rules.Frame
			for n := range s.Table.Rules.Bonus.EntryFrames {
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
				if n+1 < s.Table.Rules.Bonus.EntryFrames && (s.Bonus == nil || s.finale != nil) {
					t.Fatal("tilt bypassed the initial end-ball hold")
				}
			}
			if s.finale == nil || s.Players[0].Bonus != 12_340 || s.Players[0].Score != 0 || s.Bonus != nil || s.Rules.Frame != clock+s.Table.Rules.Bonus.EntryFrames {
				t.Fatal("tilt discarded the retained bonus or entered ordinary counting")
			}
			withTilt := s.Frame(23)
			s.tiltEntered = false
			if !bytes.Equal(withTilt.Pix, s.Frame(23).Pix) {
				t.Fatal("the tilt flag hid the retained finale text")
			}
		})
	}
}
