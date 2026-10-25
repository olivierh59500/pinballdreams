package rules

import "github.com/olivierh59500/pinballdreams/internal/ruledata"

// RelaunchPending keeps an awarded credit through the ball-end decision and
// preparation. The source consumes it when the next shot enters live play.
func (r *Engine) RelaunchPending() bool { return r.RelaunchCredits != 0 }

func (r *Engine) grantRelaunchCredit() {
	if r.defs.Relaunch.Mode == ruledata.RelaunchSingle {
		r.RelaunchCredits = 1
	} else {
		r.RelaunchCredits = int(byte(r.RelaunchCredits + 1))
		if lamp := r.Lamps[r.defs.Relaunch.Indicator]; lamp != nil {
			lamp.Players = 255
		}
	}
}

// ConsumeRelaunchCredit follows the original play-sensor boundary. Reserved
// replacements retain the pending credit until ordinary ball play resumes.
func (r *Engine) ConsumeRelaunchCredit() {
	if r.ReservedBallsActive() {
		return
	}
	if r.defs.Relaunch.Mode == ruledata.RelaunchSingle {
		r.RelaunchCredits = 0
		return
	}
	credit := byte(r.RelaunchCredits) - 1
	if int8(credit) < 0 {
		credit = 0
	}
	r.RelaunchCredits = int(credit)
	if lamp := r.Lamps[r.defs.Relaunch.Indicator]; lamp != nil {
		lamp.Players = 0
		if credit != 0 {
			lamp.Players = 255
		}
	}
}

// RefreshRelaunchIndicator is the source ball-end signed-positive test.
func (r *Engine) RefreshRelaunchIndicator() {
	if lamp := r.Lamps[r.defs.Relaunch.Indicator]; lamp != nil {
		lamp.Players = 0
		if int8(byte(r.RelaunchCredits)) > 0 {
			lamp.Players = 255
		}
	}
}
