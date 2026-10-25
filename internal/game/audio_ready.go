package game

import (
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/timing"
)

// nextSessionInput advances neither the logical clock nor the session while
// an asynchronous output context initializes. Host polls keep collecting input
// in pending; already produced PCM and notes remain untouched.
func (g *Game) nextSessionInput() (session.Input, bool) {
	if g.context != nil && !g.context.initialized() {
		if g.audio.diagnostics.enabled {
			g.audio.diagnostics.readyWaitPolls++
		}
		return session.Input{}, false
	}
	cadence := timing.Nominal
	if g.Session.Table != nil {
		cadence = timing.Table
	}
	if !g.clock.Tick(cadence) {
		return session.Input{}, false
	}
	return g.pending.take(), true
}
