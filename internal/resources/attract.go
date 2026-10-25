package resources

import (
	"fmt"

	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

// AttractMessage is an original table instruction/high-score announcement.
type AttractMessage struct {
	Definition ruledata.Announcement
	Text       []byte
	ScoreRank  int
}

// RankingText fills the shared live/attract ranking template without changing
// the extracted resource. Zero scores retain the original blank digit field.
func (m AttractMessage) RankingText(score highscore.Entry) []byte {
	text := append([]byte(nil), m.Text...)
	copy(text[4:7], score.Initials)
	value := score.Score % 10_000_000_000
	digits := "          "
	if value != 0 {
		digits = fmt.Sprintf("%10d", value)
	}
	copy(text[10:20], digits)
	return text
}

// AttractGuide pairs a supplied announcement with its highlighted lamp bank
// and source table camera target, with no executable sequencer at runtime.
type AttractGuide struct {
	Message AttractMessage
	Group   int
	Camera  int
}

type TableAttract struct {
	Roaming                 []AttractMessage
	Guided                  []AttractGuide
	SwitchHold              int
	ResetStages, ResetPaths []int
	ResetArmedStages        []int
	ResetTimers             []int
	StartCue, FirstStartCue audiodata.Command
}

func (a TableAttract) Validate() error {
	if len(a.Roaming) == 0 || len(a.Guided) > 0 && a.SwitchHold < 1 || a.StartCue.Action == audiodata.None || a.FirstStartCue.Action == audiodata.None {
		return fmt.Errorf("missing native table attract schedule")
	}
	check := func(m AttractMessage) bool {
		return len(m.Text) > 0 && m.Text[len(m.Text)-1] == 255 && m.ScoreRank >= -1 && m.ScoreRank < 4 && (m.ScoreRank < 0 || len(m.Text) >= 21)
	}
	for _, message := range a.Roaming {
		if !check(message) {
			return fmt.Errorf("invalid roaming table announcement")
		}
	}
	for _, guide := range a.Guided {
		if !check(guide.Message) || guide.Camera < 0 || guide.Camera > 276 || guide.Group == 0 {
			return fmt.Errorf("invalid guided table announcement")
		}
	}
	return nil
}
