package session

import (
	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/rules"
)

type tableAttract struct {
	assets         *resources.TableAttract
	engine         *rules.Engine
	scores         [4]highscore.Entry
	roaming, guide int
	guided         bool
	hold, target   int
	animation      *rules.PanelAnimation
}

type attractFrame struct {
	Camera int
	Upward bool
	Text   string
	Wrote  bool
}

func newTableAttract(assets *resources.TableAttract, engine *rules.Engine, scores [4]highscore.Entry) *tableAttract {
	return &tableAttract{assets: assets, engine: engine, scores: scores}
}

// Reset follows state six's instruction-list restart. Its separate camera
// target and shared pan direction survive the table reset in the original.
func (a *tableAttract) Reset(engine *rules.Engine, scores [4]highscore.Entry) {
	a.engine, a.scores = engine, scores
	a.roaming, a.guide, a.hold = 0, 0, 0
	a.guided, a.animation = false, nil
}

func (a *tableAttract) message(def resources.AttractMessage) *rules.PanelAnimation {
	if def.ScoreRank >= 0 {
		def.Text = def.RankingText(a.scores[def.ScoreRank])
	}
	return a.engine.NewAttractAnimation(def)
}

func moveTableCamera(camera, target int) int {
	if camera < target {
		return camera + 1
	}
	if camera > target {
		return camera - 1
	}
	return camera
}

// Tick executes named source roaming, guided-message and guided-hold phases.
// The caller owns the shared camera position/direction and retained text.
func (a *tableAttract) Tick(camera int, upward bool) attractFrame {
	result := attractFrame{Camera: camera, Upward: upward}
	if !a.guided {
		result.Camera, result.Upward = bounceTableCamera(camera, upward)
		a.engine.ScoreCycleTick()
		if a.animation == nil {
			a.animation = a.message(a.assets.Roaming[a.roaming])
		}
		var done bool
		result.Text, result.Wrote, done = a.animation.Step()
		if done {
			a.animation = nil
			a.roaming++
			if a.roaming == len(a.assets.Roaming) {
				a.roaming = 0
				if len(a.assets.Guided) != 0 {
					a.guided = true
					a.hold = a.assets.SwitchHold
					a.engine.ResetAttractLamps()
				}
			}
		}
		return result
	}
	result.Camera = moveTableCamera(camera, a.target)
	if a.hold != 0 {
		a.hold--
		a.engine.AttractGuidedWait()
		return result
	}
	entry := a.assets.Guided[a.guide]
	a.target = entry.Camera
	if a.animation == nil {
		a.animation = a.message(entry.Message)
	}
	var done bool
	result.Text, result.Wrote, done = a.animation.Step()
	if done {
		a.animation = nil
		a.guide++
		if a.guide == len(a.assets.Guided) {
			a.guide = 0
			a.guided = false
		} else {
			a.engine.BeginLiveFrame()
			a.engine.ResetAttractLamps()
			a.hold = a.assets.SwitchHold
			result.Camera = camera
			return result
		}
	}
	result.Camera = moveTableCamera(camera, a.target)
	a.engine.AttractGuidedMessage(entry.Group)
	return result
}

// bounceTableCamera follows the shared original attract/game-over pan helper.
// Direction is retained between calls. A legal endpoint toggles after moving;
// a supplied bottom/downward boundary can therefore produce row 277 once.
func bounceTableCamera(camera int, upward bool) (int, bool) {
	if upward {
		camera--
		if camera <= 0 {
			camera = max(0, camera)
			upward = false
		}
	} else {
		camera++
		if camera >= 276 {
			upward = true
		}
	}
	return camera, upward
}

// startTableCamera is the original state-6 pan after a player count is chosen:
// two bounce steps per wait below row 267, then one until the row-276 endpoint.
// Its caller clears direction once when entering this phase, not every tick.
func startTableCamera(camera int, upward bool) (int, bool) {
	camera, upward = bounceTableCamera(camera, upward)
	if camera < 267 {
		camera, upward = bounceTableCamera(camera, upward)
	}
	return camera, upward
}
