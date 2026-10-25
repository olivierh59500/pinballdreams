package session

import (
	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

type ballStatus struct {
	assets    *resources.BallPreparationUI
	remaining int
	blank     bool
	text      string
	mode      ballStatusMode
	score     bool
}

type ballStatusMode string

const (
	normalStatus   ballStatusMode = "normal"
	reservedStatus ballStatusMode = "reserved"
	relaunchStatus ballStatusMode = "relaunch"
)

func (b *ballStatus) advance(frame, players, player, ball int, scoreText string) string {
	b.score = false
	if b.mode == reservedStatus || b.mode == relaunchStatus {
		visible := b.blank
		if b.assets.SharedBlink {
			visible = frame&b.assets.SpecialBlinkFrames != 0
		} else {
			b.blank = !b.blank
		}
		b.text = b.assets.BlankPrompt
		if visible {
			prompt := b.assets.Reserved
			if b.mode == relaunchStatus {
				prompt = b.assets.Relaunch
			}
			text := []byte(prompt.Text)
			text[prompt.PlayerCell] = byte('1' + player)
			b.text = string(text)
		}
		return b.text
	}
	if b.assets.ScoreAlternateFrames != 0 && frame&b.assets.ScoreAlternateFrames != 0 {
		b.text = scoreText
		b.score = true
		return b.text
	}
	if b.assets.SharedBlink {
		b.blank = frame&b.assets.BlinkFrames != 0
	} else {
		b.remaining--
		if b.remaining == 0 {
			b.remaining = b.assets.BlinkFrames
			b.blank = !b.blank
		}
	}
	text := []byte(b.assets.Prompt)
	for n := range players {
		text[n] = byte('1' + n)
	}
	text[b.assets.BallCell] = byte('0' + ball)
	if b.blank {
		text[player] = ' '
	}
	b.text = string(text)
	return b.text
}

// ballPreparation owns two named countdown intervals before live spring play.
// Spring input remains live while ball motion waits for both intervals.
type ballPreparation struct {
	status    ballStatus
	phase     int
	remaining int
}

type preparationFrame struct {
	Phase, Remaining int
	Text             string
	Cue              audiodata.Command
	Done             bool
	Score            bool
}

func newBallPreparation(assets *resources.BallPreparationUI, blank bool) *ballPreparation {
	return &ballPreparation{
		status:    ballStatus{assets: assets, remaining: 1, blank: blank},
		remaining: assets.HoldFrames[0],
	}
}

func (p *ballPreparation) advance(frame, players, player, ball int, scoreText string) preparationFrame {
	result := preparationFrame{Phase: p.phase, Remaining: p.remaining}
	result.Text = p.status.advance(frame, players, player, ball, scoreText)
	result.Score = p.status.score
	p.remaining--
	if p.remaining == 0 {
		if p.phase == 0 {
			// Source dispatches this cue after the first loop's final draw and
			// decrement, before waiting for the second loop's first frame.
			result.Cue = p.status.assets.Cues[1]
			p.phase = 1
			p.remaining = p.status.assets.HoldFrames[1]
		} else {
			result.Done = true
		}
	}
	return result
}

// PreparingBall distinguishes target restoration and countdowns from live play.
func (s *Session) PreparingBall() bool {
	return s.startPhase != tableStartIdle || s.pendingNewBall || s.restoration != nil || s.preparation != nil
}

func (s *Session) prepareBallFrame(input Input) {
	s.Rules.PreparationTick()
	if s.restoration != nil {
		s.restoration.Restore()
		if !s.restoration.Pending() {
			s.restoration.Finish()
			s.restoration = nil
			s.serveNormalBall()
			s.beginBallCountdown()
		}
		return
	}
	s.pollPlayerSelection(input)
	step := s.preparation.advance(s.Rules.Frame, s.PlayerCount, s.Player, s.Ball, s.Table.ScoreText(s.Player, s.Players[s.bankPlayer].Score))
	s.writePanel(step.Text)
	if step.Score {
		s.panel.number(s.Players[s.bankPlayer].Score)
	}
	if step.Cue.Action != audiodata.None && step.Cue.Action != "" {
		s.Sound.Trigger(step.Cue)
	}
	if step.Done {
		s.ballStatus = &s.preparation.status
		s.preparation = nil
		s.launching = true
		s.bankPlayer = s.Player
		s.Engine.Locked = false
		s.spritesHidden = false
	}
}
