package session

import (
	"github.com/olivierh59500/pinballdreams/internal/highscore"
)

// ConfigureScores attaches a platform-owned directory. Headless sessions stay
// in memory unless their caller explicitly supplies storage.
func (s *Session) ConfigureScores(directory string) error {
	store := &highscore.Store{Directory: directory}
	book, err := store.Load(s.presentation.Scores.Defaults, " 0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if err != nil {
		return err
	}
	s.ScoreBook, s.scoreStore = book, store
	return nil
}

func (s *Session) endGame() {
	s.Ball = 1
	s.preparation = nil
	s.restoration = nil
	s.launching = false
	s.keyboardPausePending = false
	s.Playing = false
	s.gameOver = false
	s.gameOverFrames = 0
	s.scorePlayer = 0
	s.scoresEntered = false
	s.nextScoreEntry()
	if s.NameEntry == nil {
		s.beginScoreCycle()
	}
}

func (s *Session) beginScoreCycle() {
	s.Sound.PrepareGameOver(s.Table.ScoreEntry.GameOverLoopBytes)
	s.Player, s.bankPlayer, s.Rules.Player = 0, 0, 0
	s.gameOver = true
	s.spritesHidden = true
	s.gameOverFrames = 0
	if !s.scoresEntered {
		s.Sound.Trigger(s.Table.ScoreEntry.GameOverMusic)
	}
}

func (s *Session) nextScoreEntry() {
	scores := make([]uint64, s.PlayerCount)
	for n := range scores {
		scores[n] = s.Players[n].Score
	}
	if candidate, ok := s.ScoreBook.Qualify(s.Table.ID, scores[s.scorePlayer:]); ok {
		candidate.Player += s.scorePlayer
		// The source inserts the score before waiting for initials, retaining
		// the destination row's old letters until each character is edited.
		initials := s.ScoreBook[candidate.Table][candidate.Rank].Initials
		s.ScoreBook.Insert(candidate, initials)
		s.NameEntry = highscore.NewEditor(candidate, s.Table.ScoreEntry.Allowed)
		if !s.scoresEntered {
			s.Sound.Trigger(s.Table.ScoreEntry.Music)
		}
		s.scoresEntered = true
	}
}

func (s *Session) updateNameEntry(input Input) error {
	s.beginTableWait(input)
	entry := s.NameEntry
	// Initials wait on the shared clock without running table presentation.
	s.Rules.BeginLiveFrame()
	entry.Step(input.Text, input.Backspace && !s.previous.Backspace || input.Back && !s.previous.Back)
	switch input.InitialsEdit {
	case InitialsPrevious, InitialsNext:
		entry.Cycle(0)
		delta := 1
		if input.InitialsEdit == InitialsPrevious {
			delta = -1
		}
		entry.Cycle(delta)
	case InitialsChoose:
		// A labelled touch confirmation commits the character already shown
		// in the touch preview, including its initial A selection.
		entry.Cycle(0)
		entry.Choose()
	case InitialsErase:
		entry.Backspace()
	}
	if input.Left && !s.previous.Left {
		entry.Cycle(-1)
	}
	if input.Right && !s.previous.Right {
		entry.Cycle(1)
	}
	if input.Launch && !s.previous.Launch {
		entry.Choose()
	}
	if s.startPressed(input) {
		entry.Finish()
	}
	row := &s.ScoreBook[entry.Candidate.Table][entry.Candidate.Rank]
	row.Initials = entry.RowInitials(row.Initials)
	s.writePanel(s.scoreEntryText())
	if entry.Done() {
		s.scorePlayer = entry.Candidate.Player + 1
		s.NameEntry = nil
		if s.scoreStore != nil {
			if err := s.scoreStore.Save(s.ScoreBook); err != nil {
				return err
			}
		}
		s.nextScoreEntry()
		if s.NameEntry == nil {
			s.beginScoreCycle()
		}
	}
	return nil
}

func (s *Session) scoreEntryText() string {
	ui := s.Table.ScoreEntry
	return s.NameEntry.Line(ui.Prompt, ui.PlayerCell, ui.InitialsCell)
}
