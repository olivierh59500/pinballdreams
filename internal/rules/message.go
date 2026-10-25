package rules

import "github.com/olivierh59500/pinballdreams/internal/ruledata"

// announcement is a native translation of the source score-display sequencer.
// Scroll uses sixteenth-character units; blink/reveal counters retain byte wrap.
type announcement struct {
	ID, TextID                 int
	Flags, Period, Repeats     byte
	Speed                      int16
	Text                       []byte
	Started, Phase, New        bool
	Counter, Remaining, Reveal byte
	Scroll, Time               int16
	Buffer                     [21]byte
	Display                    [20]byte
	Done                       bool
	MaskIndex                  int
	Masks                      [][20]byte
	MasksEnabled               bool
}

func newAnnouncement(def ruledata.Announcement, text []byte, phase bool, previous string) *announcement {
	m := &announcement{ID: def.ID, TextID: def.Text, Flags: def.Flags, Period: def.Period, Repeats: def.Repeats, Speed: def.Speed, Phase: phase, Text: text, Masks: def.Masks, MasksEnabled: def.MasksEnabled}
	m.Buffer[20] = 255
	for i := range m.Display {
		m.Display[i] = ' '
	}
	copy(m.Display[:], []byte(previous))
	return m
}

func characterAt(text []byte, offset int) byte {
	if offset < 0 || offset >= len(text) {
		return 255
	}
	return text[offset]
}

// step returns whether a glyph was written. Source early returns preserve the
// previous display while a scrolling line is still outside the visible window.
func (m *announcement) step() (string, bool) {
	if !m.Started {
		m.Started = true
		m.Counter = 1
		m.Remaining = m.Repeats
		m.Scroll = 0
		m.Time = m.Speed
		m.Reveal = ' '
		m.New = true
	}
	count, position, offset := 20, 0, 0
	text := m.Text
	if m.Flags&messageScroll != 0 {
		offset = int(uint16(m.Scroll)>>4) - 20
		m.Scroll += m.Speed
		if m.Scroll < 0 {
			m.finish()
			return "", false
		}
		if offset < 0 {
			position = -offset
			count -= position
			offset = 0
			if count <= 0 {
				return "", false
			}
		} else if characterAt(text, offset) == 255 {
			m.finish()
			return "", false
		}
	}
	// The second source mode jumps directly to the next flag. Its historical
	// left-scroll routine is unreachable in this supplied release.
	if m.Flags&messageBlink != 0 {
		m.Counter--
		if m.Counter == 0 {
			m.Phase = !m.Phase
			m.Counter = m.Period
			m.Remaining--
			if m.Remaining == 0 {
				m.finish()
				return "", false
			}
		}
		if m.Phase {
			text = []byte("                    \xff")
			offset = 0
		}
	}
	if m.Flags&messageShort != 0 && characterAt(text, offset+18) == 255 {
		m.finish()
		return "", false
	}
	if m.Flags&messageReveal != 0 {
		if m.Reveal > 'Z' {
			m.Time--
			if m.Time == 0 {
				m.finish()
				return "", false
			}
		} else {
			m.Counter--
			if m.Counter == 0 {
				m.Counter = m.Period
				if m.Period == 0 {
					m.finish()
					return "", false
				}
				pending := false
				for i := range count {
					c := characterAt(text, offset+i)
					if m.Reveal <= c {
						c = m.Reveal
						pending = true
					}
					m.Buffer[i] = c
				}
				m.Buffer[count] = 255
				m.Reveal++
				if !pending {
					m.Time--
					if m.Time == 0 {
						m.finish()
						return "", false
					}
				}
			}
		}
		text = m.Buffer[:]
		offset = 0
	}
	if m.Flags&messageMasks != 0 && m.MasksEnabled {
		if m.MaskIndex == len(m.Masks) {
			m.Time--
			if m.Time == 0 {
				m.finish()
				return "", false
			}
		} else {
			for i := range 20 {
				c := m.Masks[m.MaskIndex][i]
				if c == ' ' {
					c = characterAt(text, offset+i)
				}
				m.Buffer[i] = c
			}
			text, offset = m.Buffer[:], 0
			m.Counter--
			if m.Counter == 0 {
				m.Counter = m.Period
				m.MaskIndex++
			}
		}
	}
	wrote := false
	for i := range count {
		c := characterAt(text, offset+i)
		if c == 255 {
			break
		}
		m.Display[position+i] = c
		wrote = true
	}
	return string(m.Display[:]), wrote
}

func (m *announcement) finish() { m.Done = true; m.New = false }
