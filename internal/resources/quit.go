package resources

import (
	"fmt"
	"strings"
)

// QuitUI declares the supplied question and the original exit fade policy.
type QuitUI struct {
	Prompt                  string
	ConfirmCell, CancelCell int
	FadeTarget, FadeStep    byte
	PaletteFrames           int
}

func (q QuitUI) Validate() error {
	choices := strings.Index(q.Prompt, "(Y/N)")
	if len(q.Prompt) != 20 || choices < 0 || q.ConfirmCell != choices+1 || q.CancelCell != choices+3 || q.FadeTarget != 0 || q.FadeStep < 1 || q.PaletteFrames < 1 || q.PaletteFrames > 16 {
		return fmt.Errorf("invalid native quit presentation")
	}
	return nil
}
