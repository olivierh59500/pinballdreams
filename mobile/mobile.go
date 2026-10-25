//go:build android

// Package mobile is the thin Android bridge to the native Go game.
package mobile

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/hajimehoshi/ebiten/v2/mobile/ebitenmobileview"
	"github.com/olivierh59500/pinballdreams/internal/game"
	"github.com/olivierh59500/pinballdreams/internal/platforminput"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/timing"
)

var host struct {
	sync.Mutex
	ready, pause, back bool
	orientationChanged bool
	game               *game.Game
	saveDirectory      string
	cancelledTouches   []int
	diagnostics        bool
	touchEvents        []touchEvent
}

type touchEvent struct {
	action, id int
	x, y       float32
}

var navigationTrace struct {
	previous session.NavigationState
	ready    bool
}

var touchQueue = platforminput.NewTouchQueue(func(event platforminput.Touch) {
	ebitenmobileview.UpdateTouchesOnAndroid(int(event.Action), event.ID, event.X, event.Y)
})
var touchSnapshot []ebiten.TouchID
var observedTouches []int

func init() {
	ebiten.SetTPS(timing.HostTicksPerSecond)
	ebiten.SetScreenFilterEnabled(false)
	enginemobile.SetGame(&hostGame{})
}

// Configure defers game creation until Android has initialized its view.
func Configure(saveDirectory string) {
	host.Lock()
	host.ready = true
	host.saveDirectory = saveDirectory
	host.Unlock()
}

// SetDiagnosticsEnabled enables semantic navigation traces in debug APKs.
func SetDiagnosticsEnabled(enabled bool) {
	host.Lock()
	host.diagnostics = enabled
	host.Unlock()
}

// RecordTouch copies platform lifecycle events for diagnostics only. Mapping
// and delivery of gameplay controls still belong to the shared Go host.
func RecordTouch(action, id int, x, y float32) {
	host.Lock()
	if host.diagnostics {
		host.touchEvents = append(host.touchEvents, touchEvent{action, id, x, y})
	}
	host.Unlock()
}

// Suspend queues the pause on the game thread.
func Suspend() {
	host.Lock()
	host.pause = true
	touchQueue.CancelAll()
	host.Unlock()
}

// OrientationChanged queues a platform viewport transition. Existing tables,
// including attract, pause before remapped contacts can release the spring.
func OrientationChanged() {
	host.Lock()
	host.orientationChanged = true
	touchQueue.CancelAll()
	host.Unlock()
}

// CancelTouch queues an Android-cancelled contact. Ebitengine 2.10.2 retains
// ACTION_CANCEL contacts; retire them after silently cancelling the spring,
// so cancellation cannot become a launch or absorb the next resume gesture.
func CancelTouch(id int) {
	host.Lock()
	host.cancelledTouches = append(host.cancelledTouches, id)
	touchQueue.Cancel(id)
	host.Unlock()
}

// UpdateTouch normalizes Android lifecycle packets in Go. Presses and motion
// keep their ordinary delivery; an unobserved release waits for a game poll.
func UpdateTouch(action, id int, x, y, density float32) {
	if density <= 0 {
		return
	}
	var kind platforminput.TouchAction
	switch action {
	case 0, 5:
		kind = platforminput.TouchDown
	case 1, 6:
		kind = platforminput.TouchUp
	case 2:
		kind = platforminput.TouchMove
	default:
		return
	}
	touchQueue.Push(platforminput.Touch{Action: kind, ID: id, X: float64(x) / float64(density), Y: float64(y) / float64(density)})
}

// Back queues Android's system back gesture on the game thread.
func Back() { host.Lock(); host.back = true; host.Unlock() }

// Dummy keeps a stable binding entry for ebitenmobile.
func Dummy() {}

type hostGame struct{}

func (*hostGame) Update() error {
	host.Lock()
	ready, pause, back := host.ready, host.pause, host.back
	rotated := host.orientationChanged
	saveDirectory := host.saveDirectory
	cancelled := host.cancelledTouches
	diagnostics, touches := host.diagnostics, host.touchEvents
	host.touchEvents = nil
	host.cancelledTouches = nil
	host.pause, host.back = false, false
	host.orientationChanged = false
	host.Unlock()
	if host.game == nil && ready {
		var err error
		host.game, err = game.NewWithStorage(nil, true, saveDirectory)
		if err != nil {
			return err
		}
	}
	if host.game == nil {
		return nil
	}
	host.game.SetAudioDiagnostics(diagnostics)
	if pause {
		host.game.Suspend()
	}
	if rotated {
		host.game.OrientationChanged()
	}
	for _, id := range cancelled {
		host.game.CancelTouch(ebiten.TouchID(id))
	}
	if back {
		host.game.Back()
	}
	if diagnostics {
		for _, event := range touches {
			fmt.Printf("pinball-touch action=%d id=%d x=%g y=%g\n", event.action, event.id, event.x, event.y)
		}
	}
	err := host.game.Update()
	touchSnapshot = ebiten.AppendTouchIDs(touchSnapshot[:0])
	observedTouches = observedTouches[:0]
	for _, id := range touchSnapshot {
		observedTouches = append(observedTouches, int(id))
	}
	touchQueue.Observe(observedTouches)
	if diagnostics {
		if snapshot, ready := host.game.PollAudioDiagnostics(time.Now()); ready {
			if encoded, err := json.Marshal(snapshot); err == nil {
				fmt.Printf("pinball-audio-software %s\n", encoded)
			}
		}
		state := host.game.Session.NavigationState()
		previous := navigationTrace.previous
		// The clock stamps changes but must not turn this into a per-frame log.
		previous.Clock = state.Clock
		if !navigationTrace.ready || previous != state {
			fmt.Printf("pinball-navigation screen=%s table=%d clock=%d playing=%t paused=%t exit=%q stopped=%t fade_complete=%t blocked=%t\n", state.Screen, state.Table, state.Clock, state.Playing, state.Paused, state.ExitPhase, state.AudioStopped, state.FadeComplete, state.ExitBlocked)
			navigationTrace.previous, navigationTrace.ready = state, true
		}
	}
	return err
}
func (*hostGame) Draw(screen *ebiten.Image) {
	if host.game != nil {
		host.game.Draw(screen)
	}
}
func (*hostGame) Layout(width, height int) (int, int) {
	if host.game != nil {
		return host.game.Layout(width, height)
	}
	return 320, 256
}
