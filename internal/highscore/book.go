// Package highscore owns native rankings, initials entry and score persistence.
package highscore

import "fmt"

const (
	Tables = 4
	Ranks  = 4
)

type Entry struct {
	Initials string
	Score    uint64
}

type Book [Tables][Ranks]Entry

type Candidate struct {
	Table, Rank, Player int
	Score               uint64
}

// Qualify follows the supplied game's player order and strict comparison.
// A tie keeps the existing entry. The caller continues after the returned
// player once those initials have been committed, preserving player order.
func (b *Book) Qualify(table int, scores []uint64) (Candidate, bool) {
	for player, score := range scores {
		for rank, entry := range b[table] {
			if score > entry.Score {
				return Candidate{Table: table, Rank: rank, Player: player, Score: score}, true
			}
		}
	}
	return Candidate{}, false
}

func (b *Book) Insert(candidate Candidate, initials string) {
	rows := &b[candidate.Table]
	for rank := Ranks - 1; rank > candidate.Rank; rank-- {
		rows[rank] = rows[rank-1]
	}
	rows[candidate.Rank] = Entry{Initials: initials, Score: candidate.Score}
}

func (b *Book) Validate(allowed string) error {
	for _, table := range b {
		for rank, entry := range table {
			if len(entry.Initials) != 3 || entry.Score >= 10_000_000_000_000_000 || rank > 0 && entry.Score > table[rank-1].Score {
				return fmt.Errorf("invalid high-score entry")
			}
			for _, c := range []byte(entry.Initials) {
				found := false
				for _, valid := range []byte(allowed) {
					if c == valid {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("invalid high-score initial")
				}
			}
		}
	}
	return nil
}

// Digits reproduces the ten displayed decimal places and blank leading zeroes.
func Digits(score uint64) string {
	if score%10_000_000_000 == 0 {
		return "          "
	}
	return fmt.Sprintf("%10d", score%10_000_000_000)
}
