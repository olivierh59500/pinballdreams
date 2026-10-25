package session

import "github.com/olivierh59500/pinballdreams/internal/frontend"

// tableSelection retains the chosen table while the existing scene fades.
// Resource loading starts after the final completely black presentation frame.
type tableSelection struct {
	table, remaining int
	loading          *frontend.Loading
}

func (s *Session) selectFromMenu(table int) {
	if s.Scores != nil {
		// The original ranking key matcher clears the inherited cadence
		// before it dispatches a function-key table choice.
		s.Sound.SetRetriggerPhase([16]byte{})
	}
	s.selection = &tableSelection{table: table, remaining: s.presentation.Menu.ExitFrames}
	s.Menu.Hide()
	if s.Scores != nil {
		s.Scores.Hide()
	}
	s.footer.Hide()
}

func (s *Session) updateTableSelection(input Input) error {
	if s.selection.remaining == 0 && s.selection.loading != nil && s.selection.loading.Ready() {
		if err := s.Select(s.selection.table - 1); err != nil {
			return err
		}
		s.beginTableEntry()
		// The next original wait samples table input and replay before
		// raising its palette; it no longer belongs to the menu module.
		s.advanceTableEntry(input)
		return nil
	}
	s.advanceAudio()
	if s.selection.remaining == 0 {
		if s.selection.loading == nil {
			s.selection.loading = frontend.NewLoading(&s.presentation.Loading, s.selection.table-1)
		}
		s.selection.loading.Tick()
		return nil
	}
	s.Menu.Tick()
	if s.Scores != nil {
		s.Scores.Tick()
	}
	s.footer.Tick()
	s.selection.remaining--
	return nil
}
