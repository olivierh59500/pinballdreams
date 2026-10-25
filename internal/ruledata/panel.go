package ruledata

// PanelUI describes the source live display's idle announcement cycle.
type PanelUI struct {
	InitialDelay, RepeatDelay int
	Idle                      []int
	Bindings                  []PanelBinding
	Special                   PanelSpecial
}

// PanelSpecial describes the supplied saved-award running-value display.
type PanelSpecial struct {
	Stage, Bank, Text, Priority, ClockMask int
	Delta                                  uint64
	Add, ClearAtLowZero                    bool
}

// PanelBinding resolves a mutable source status field into a native value.
type PanelBinding struct {
	Text, Position, Digits int
	Kind                   string
	Bank                   int
	ZeroPad                bool
}
