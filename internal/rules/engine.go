package rules

import (
	"slices"

	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/ruledata"
)

type lampState struct {
	Flags, Players byte
	Blink, Phase   bool
}
type groupState struct {
	Timer     int
	Direction bool
}
type stageState struct {
	Players byte
	Timer   int
}
type action struct {
	ID  int
	Arm bool
}

// Player retains the actor's booster and advance records between turns.
type Player struct {
	Boosters, Advances int
}

// Engine owns high-level table records, feature programs and lamp animation.
// It never interprets the original CPU instructions or custom-chip registers.
type Engine struct {
	sequence                  *sequence
	directMessage             bool
	Layer                     int
	locks                     lockState
	textOverrides             map[int][]byte
	Locked                    bool
	tilted                    bool
	capture                   *capture
	captureContact            bool
	OnCapture                 func(int, int, int16, int16)
	announcement              *announcement
	messagePhase              bool
	messageBuffer             [21]byte
	jackpots                  map[int]uint64
	collectedJackpots         map[int]uint64
	awardSources, savedAwards map[int]uint64
	OnTargetChange            func(int, bool)
	OnTargetLayerChange       func(int, int, bool)
	targetCooldown            map[int]int
	resetTargets              []int
	resetTargetTimer          int
	Banks                     func() (uint64, uint64)
	ActorPlayer               func() int
	PlayerCount               int
	BanksFor                  func(int) (uint64, uint64)
	defs                      *definitions
	Player, Frame             int
	Players                   [8]Player
	// Current-ball features belong to the table, independently of the actor
	// selected for scores and the player mask selected for lamps/targets.
	BonusMultiplier, RelaunchCredits int
	HoldBonus                        bool
	Lamps                            map[int]*lampState
	groups                           map[int]*groupState
	stages                           map[int]*stageState
	counters                         map[int]*[8]int
	timers                           map[int]int
	hits                             map[int]bool
	collected                        map[int]byte
	queue                            []action
	flashGroup, flashTimer           int
	skill                            bool
	message                          string
	messagePriority                  int
	PanelText                        func() string
	Unsupported                      map[int]bool
	OnAward                          func(uint64, uint64)
	OnScoreActivity                  func()
	OnPanelNumber                    func(uint64)
	OnRetriggerPhase                 func([16]byte)
	OnSound                          func(audiodata.Command)
	OnExtraBall                      func()
	artwork                          *artworkState
}

func New(table *resources.Table) (*Engine, error) {
	d, err := decode(table)
	if err != nil {
		return nil, err
	}
	return fromDefinitions(d), nil
}

func fromDefinitions(d *definitions) *Engine {
	r := &Engine{defs: d, Lamps: make(map[int]*lampState), groups: make(map[int]*groupState), stages: make(map[int]*stageState), counters: make(map[int]*[8]int), timers: make(map[int]int), hits: make(map[int]bool), collected: make(map[int]byte), Unsupported: make(map[int]bool)}
	r.targetCooldown = make(map[int]int)
	r.PlayerCount = 1
	r.locks.Slots = make([]bool, len(d.Locks.Slots))
	r.textOverrides = make(map[int][]byte)
	r.messageBuffer[20] = 255
	r.jackpots = make(map[int]uint64)
	r.collectedJackpots = make(map[int]uint64)
	r.awardSources, r.savedAwards = make(map[int]uint64), make(map[int]uint64)
	for id, bank := range d.Jackpots {
		r.jackpots[id] = bank.Initial
		r.collectedJackpots[id] = bank.Initial
		r.awardSources[id], r.savedAwards[id] = bank.AwardSourceInitial, bank.SavedAwardInitial
	}
	for id, target := range d.Targets {
		r.collected[id] = target.InitialPlayers
		r.targetCooldown[id] = target.InitialCooldown
	}
	for id, lamp := range d.Lamps {
		r.Lamps[id] = &lampState{Flags: lamp.Flags, Blink: lamp.InitialBlink != 0, Players: lamp.InitialPlayers, Phase: lamp.InitialPhase != 0}
	}
	for _, group := range d.Groups {
		r.groups[group.ID] = &groupState{Timer: group.InitialTimer, Direction: group.InitialDirection}
	}
	for id, stage := range d.Stages {
		state := &stageState{Players: stage.InitialPlayers}
		if stage.Kind == ruledata.Repeatable && !stage.Permanent && slices.Contains(d.Table.Attract.ResetStages, id) {
			state.Players = 255
		}
		r.stages[id] = state
	}
	r.BonusMultiplier = 1
	return r
}

// Reset starts a new game while retaining the immutable decoded rule records.
func (r *Engine) Reset() {
	fresh := fromDefinitions(r.defs)
	// Starting a game resets player records, but the shared animation clocks
	// continue through attract mode and the source's skill-shot selection.
	fresh.Frame, fresh.groups = r.Frame, r.groups
	fresh.messagePhase, fresh.messageBuffer = r.messagePhase, r.messageBuffer
	// Starting a new game keeps the ranking templates last refreshed by the
	// complete table reset, including their saved initials and scores.
	for _, message := range r.defs.Table.Attract.Roaming {
		if message.ScoreRank >= 0 {
			if text := r.textOverrides[message.Definition.Text]; text != nil {
				fresh.textOverrides[message.Definition.Text] = text
			}
		}
	}
	for id, lamp := range r.Lamps {
		fresh.Lamps[id].Flags = lamp.Flags &^ lampFlash
		fresh.Lamps[id].Blink = lamp.Blink
		fresh.Lamps[id].Phase = lamp.Phase
	}
	for id, stage := range r.defs.Stages {
		if stage.Permanent || !slices.Contains(r.defs.Table.Attract.ResetStages, id) {
			*fresh.stages[id] = *r.stages[id]
		}
	}
	fresh.OnAward, fresh.OnSound, fresh.OnExtraBall, fresh.Banks = r.OnAward, r.OnSound, r.OnExtraBall, r.Banks
	fresh.OnScoreActivity = r.OnScoreActivity
	fresh.OnPanelNumber = r.OnPanelNumber
	fresh.OnRetriggerPhase = r.OnRetriggerPhase
	fresh.ActorPlayer = r.ActorPlayer
	fresh.PlayerCount, fresh.BanksFor = r.PlayerCount, r.BanksFor
	fresh.PanelText = r.PanelText
	fresh.OnTargetChange = r.OnTargetChange
	fresh.OnTargetLayerChange = r.OnTargetLayerChange
	fresh.OnCapture = r.OnCapture
	*r = *fresh
}

func (r *Engine) mask() byte { return 1 << uint(r.Player) }

// Presentation sequences have eight ownership lanes and wrap after the last
// selected player. The final eight-player encore still has an empty display
// mask, while these sequence services use the first ownership lane.
func (r *Engine) presentationMask() byte { return 1 << uint(r.Player&7) }

// actorFeatures follows the score-bank owner. During target restoration and
// readiness, lamp/target masks already select the next player while the actor
// bank and its booster/advance records still belong to the outgoing player.
func (r *Engine) actorFeatures() *Player {
	return &r.Players[r.actorIndex()]
}

func (r *Engine) actorIndex() int {
	player := r.Player
	if r.ActorPlayer != nil {
		player = r.ActorPlayer()
	}
	return player
}

// NewBall prepares a ball immediately for isolated rule callers. Live sessions
// pace TargetRestoration across presentation frames before calling Finish.
func (r *Engine) NewBall(player int) {
	restoration := r.BeginTargetRestoration(player)
	for restoration.Pending() {
		restoration.Restore()
	}
	restoration.Finish()
}

func (r *Engine) initializeBall() {
	r.tilted = false
	r.sequence = nil
	r.directMessage = false
	r.clearLocks()
	r.Layer = 0
	r.announcement = nil
	r.message = ""
	r.messagePriority = 0
	r.capture = nil
	r.Locked = false
	r.BonusMultiplier = 1
	r.queue = nil
	r.flashTimer = 0
	r.hits = make(map[int]bool)
	r.resetTargets = nil
	r.resetTargetTimer = 0
	r.resetBallLamps()
	// Referenced awards can be absent from the reset list. In particular,
	// Ignition's million-point stage retains its supplied player byte.
	for _, id := range r.defs.Table.Attract.ResetStages {
		state := r.stages[id]
		if state == nil {
			continue
		}
		if r.defs.Stages[id].Flags&(stageCollectedTimer|stageArmedTimer) != 0 {
			state.Timer = 0
		}
		if r.defs.Stages[id].RetainOnBall {
			continue
		}
		if r.defs.Stages[id].Kind == ruledata.Repeatable {
			state.Players = 255
		} else {
			state.Players = 0
		}
	}
	for id, path := range r.defs.Paths {
		if path.ResetOnBall {
			delete(r.counters, id)
		}
	}
	r.skill = len(r.defs.SkillLamps) > 0
	if r.skill {
		selected := r.defs.SkillLamps[(r.Frame&7)%len(r.defs.SkillLamps)]
		r.Lamps[selected].Flags |= lampFlash
		r.Lamps[selected].Players = 0
		r.Lamps[selected].Phase = false
	}
	r.invoke(r.defs.BallStartPath)
	r.initializeGates()
}

// resetBallLamps clears temporary collection ownership across the table.
// Shared animation and flash-phase bytes survive the ball transition.
func (r *Engine) resetBallLamps() {
	for _, lamp := range r.Lamps {
		lamp.Flags &^= lampFlash
		if lamp.Flags&lampPersistent == 0 {
			lamp.Players = 0
		}
	}
}

// Tick is a convenience update for isolated rule callers. Live sessions split
// these phases around their motion and contact services. Attract rendering
// keeps its independent clock; tilted callers retain presentation-only work.
func (r *Engine) Tick(attract, tilted, rotate bool) {
	r.BeginLiveFrame()
	if attract {
		r.advanceAnnouncement()
		r.Animate(true)
		r.advanceArtwork(true, tilted)
		return
	}
	r.BeforeContacts()
	if !tilted {
		r.FinishLiveFrame(rotate)
		r.FinishCapture(false)
		return
	}
	r.Animate(false)
	r.advanceSkill()
	r.advanceArtwork(false, true)
	r.advanceLampTimers()
	r.sequential()
	r.flash()
	if rotate {
		r.rotate()
	}
	r.decrementTargetCooldowns()
	r.FinishCapture(false)
}

func (r *Engine) advanceAnnouncement() {
	if r.sequence == nil && r.directMessage {
		r.message = ""
		r.directMessage = false
	}
	if m := r.announcement; m != nil && r.sequence == nil {
		// Status values can change after an announcement starts. Each reveal
		// uses the current shared template, retaining only its animation state.
		m.Text = r.messageText(m.TextID)
		line, wrote := m.step()
		r.messagePhase = m.Phase
		r.messageBuffer = m.Buffer
		if m.Done {
			r.announcement = nil
			r.message = ""
			r.messagePriority = 0
		} else if wrote {
			r.message = line
		}
	}
}

func (r *Engine) releaseCapture() {
	r.releaseCaptureAfterCamera(false)
}

func (r *Engine) releaseCaptureAfterCamera(cameraTransition bool) {
	if r.capture != nil && r.capture.Role == ruledata.TrapObject && !r.locks.Active && !cameraTransition && r.announcement == nil && len(r.queue) == r.capture.QueueBase {
		r.capture = nil
		r.captureContact = false
		r.Locked = false
		r.sound(r.defs.CaptureReleaseSound)
	}
}

func (r *Engine) decrementTargetCooldowns() {
	for _, group := range r.defs.TargetGroups {
		if group.Layers&(1<<uint(r.Layer)) == 0 {
			continue
		}
		for _, id := range group.Targets {
			if r.defs.Targets[id].Kind != 7 {
				break
			}
			if r.targetCooldown[id] > 0 {
				r.targetCooldown[id]--
			}
		}
	}
}

// Animate advances the original shared lamp cadence independently of rules.
func (r *Engine) Animate(attract bool) {
	for _, group := range r.defs.Groups {
		state := r.groups[group.ID]
		if state.Timer != 0 {
			state.Timer--
			continue
		}
		state.Timer = group.Period
		for index, id := range group.Lamps {
			lamp := r.Lamps[id]
			if !attract && lamp.Flags&lampAnimateDuringPlay == 0 {
				break
			}
			if lamp.Flags&lampBlink != 0 {
				lamp.Blink = !lamp.Blink
				// Flags are reloaded for every linked lamp. Some Nightmare
				// chains deliberately end in a node with a different mode.
				if index+1 == len(group.Lamps) {
					state.Timer = int(byte(state.Timer - 1))
				}
				continue
			}
			if lamp.Flags&lampChase != 0 {
				found := false
				for _, next := range group.Lamps[index:] {
					node := r.Lamps[next]
					if node.Blink != state.Direction {
						node.Blink = state.Direction
						found = true
						break
					}
				}
				if !found {
					state.Direction = !state.Direction
				}
			}
			break
		}
	}
}

func (r *Engine) rotate() {
	for _, id := range r.defs.Rotating {
		group := r.group(id)
		if group == nil || len(group.Lamps) == 0 {
			continue
		}
		first := r.Lamps[group.Lamps[0]].Players
		for i := 0; i+1 < len(group.Lamps); i++ {
			r.Lamps[group.Lamps[i]].Players = r.Lamps[group.Lamps[i+1]].Players
		}
		r.Lamps[group.Lamps[len(group.Lamps)-1]].Players = first
	}
}

func (r *Engine) sequential() {
	mask := r.presentationMask()
	for _, id := range r.defs.Sequential {
		group := r.group(id)
		if group == nil {
			continue
		}
		first := true
		for _, lampID := range group.Lamps {
			lamp := r.Lamps[lampID]
			if first && r.defs.SequentialMarkCollected {
				lamp.Flags |= lampBlink
			}
			if first && lamp.Players&mask != 0 {
				continue
			}
			if first {
				lamp.Flags |= lampBlink
				first = false
			} else {
				lamp.Flags &^= lampBlink
				lamp.Players &^= mask
				lamp.Blink = false
			}
		}
	}
}

func (r *Engine) completeGroups() {
	if r.flashTimer != 0 {
		return
	}
	for _, group := range r.defs.Groups {
		if len(group.Lamps) == 0 || r.Lamps[group.Lamps[0]].Flags&lampExcludedFromBank != 0 {
			continue
		}
		complete := true
		for index, id := range group.Lamps {
			if r.Lamps[id].Players&r.mask() == 0 {
				complete = false
				break
			}
			if r.OnRetriggerPhase != nil {
				var phase [16]byte
				if index+1 < len(group.Lamps) {
					phase = r.defs.Lamps[group.Lamps[index+1]].RetriggerPhase
				}
				r.OnRetriggerPhase(phase)
			}
		}
		if !complete {
			continue
		}
		for _, id := range group.Lamps {
			r.Lamps[id].Players &^= r.mask()
			r.Lamps[id].Flags &^= lampFlash
		}
		r.flashGroup, r.flashTimer = group.ID, 50
		r.feature(group.ID)
		return
	}
}

func (r *Engine) flash() {
	if r.flashTimer == 0 {
		return
	}
	if r.OnRetriggerPhase != nil {
		r.OnRetriggerPhase([16]byte{})
	}
	r.flashTimer--
	group := r.group(r.flashGroup)
	if group == nil {
		return
	}
	phase := r.flashTimer&1 == 0
	for _, id := range group.Lamps {
		lamp := r.Lamps[id]
		if r.flashTimer == 0 {
			lamp.Flags &^= lampFlash
		} else {
			lamp.Flags |= lampFlash
			// Linked neighbors use opposite phases in a collected bank.
			lamp.Phase = phase
			phase = !phase
		}
	}
}

func (r *Engine) expireTimedStages() {
	if r.Locked && r.defs.FreezeTimedStages {
		return
	}
	for _, id := range r.defs.TimedStages {
		s := r.stages[id]
		stage := r.defs.Stages[id]
		if s.Timer > 0 {
			s.Timer--
			if s.Timer > 0 {
				continue
			}
			s.Players &^= r.mask()
		}
		r.setLamp(stage.OffLamp, false)
	}
}

func (r *Engine) advanceLampTimers() {
	mask := r.presentationMask()
	for _, timer := range r.defs.Timers {
		remaining := r.timers[timer.ID]
		if remaining == 0 {
			remaining = timer.Period
			if group := r.group(timer.Group); group != nil {
				last := 0
				for _, id := range group.Lamps {
					if r.Lamps[id].Players&mask == 0 {
						break
					}
					last = id
				}
				if last != 0 {
					r.Lamps[last].Players &^= mask
					r.sound(timer.Sound)
				}
			}
			if timer.Program != 0 && r.Player < len(r.Players) {
				counter := r.counter(timer.Program)
				if counter[r.Player] > 0 {
					counter[r.Player]--
				}
			}
		}
		r.timers[timer.ID] = remaining - 1
	}
}

func (r *Engine) group(id int) *Group {
	return r.defs.GroupsByID[id]
}
func (r *Engine) setLamp(id int, active bool) {
	if lamp := r.Lamps[id]; lamp != nil {
		if active {
			lamp.Players |= r.mask()
		} else {
			lamp.Players &^= r.mask()
		}
	}
}

func (r *Engine) sound(id int) {
	if cue, ok := r.defs.Sounds[id]; ok && r.OnSound != nil {
		r.OnSound(cue)
	}
}

func (r *Engine) award(points, bonus uint64) {
	r.addToBanks(points, bonus)
	r.scoreActivity()
}

func (r *Engine) scoreActivity() {
	if r.OnScoreActivity != nil {
		r.OnScoreActivity()
	}
}

func (r *Engine) panelNumber(value uint64) {
	if r.OnPanelNumber != nil {
		r.OnPanelNumber(value)
	}
}

// addToBanks also serves bonus counting, whose decimal increments leave the
// idle announcement timer alone. Contact/feature awards use award instead.
func (r *Engine) addToBanks(points, bonus uint64) {
	if r.OnAward != nil {
		r.OnAward(points, bonus)
	}
}

// Hit applies an object's extracted reward, indicator and gate transition.
func (r *Engine) Hit(id int, invoke bool) {
	object, ok := r.defs.Objects[id]
	if !ok {
		return
	}
	r.event(object.Event, invoke)
	r.setLamp(object.Lamp, true)
	r.applyObjectGate(object)
}

func (r *Engine) applyObjectGate(object ruledata.Object) {
	if object.Role == ruledata.GateEnterObject || object.Role == ruledata.GateExitObject {
		if _, ok := r.defs.Targets[object.Gate]; ok {
			r.collected[object.Gate] = object.GatePlayers
			r.paintTarget(object.Gate)
			if r.OnTargetChange != nil {
				r.OnTargetChange(object.Gate, object.GatePlayers&r.mask() != 0)
			}
		}
	}
}

func (r *Engine) event(id int, invoke bool) {
	if event, ok := r.defs.Events[id]; ok {
		r.award(event.Points, event.Bonus)
		r.sound(event.Sound)
		if invoke {
			r.feature(id)
		}
	}
}

// Sensors scans ordered rectangles once per entry, retaining exit latches.
func (r *Engine) Sensors(x, y int) {
	r.scanSensors(x, y, true)
}

// SensorContacts applies a native visible-ball contact result to rollovers.
// Ordinary passage rectangles continue to use their own entry/exit latches.
func (r *Engine) SensorContacts(x, y int, rollovers bool) {
	r.scanSensors(x, y, rollovers)
}

// SpringSensorContacts runs passage effects used by the source launch loop.
// Geometry and capture/feature declarations still change, while ordinary
// contact awards, sounds and collected lamps wait for the live game services.
func (r *Engine) SpringSensorContacts(x, y int, rollovers bool) {
	r.scanSensorPhase(x, y, rollovers, false)
}

// SensorActive exposes the entry latch used by spring and play transitions.
func (r *Engine) SensorActive(id int) bool { return r.hits[id] }

func (r *Engine) HasRollovers() bool { return len(r.defs.Layers[r.Layer].ExtraSensors) != 0 }

// scanSensors accepts the rollover-contact result separately from passages.
// Disabled rollovers clear their entry latches, allowing a later fresh contact.
func (r *Engine) scanSensors(x, y int, rollovers bool) {
	r.scanSensorPhase(x, y, rollovers, true)
}

func (r *Engine) scanSensorPhase(x, y int, rollovers, live bool) {
	if r.Locked && r.defs.FreezeSensors {
		return
	}
	// Passage bounds use unsigned sixteen-bit coordinates. A ball above the
	// table wraps its center rather than entering the lower-Y early stop.
	x, y = int(uint16(x)), int(uint16(y))
	layer := r.defs.Layers[r.Layer]
	r.sensorList(layer.Sensors, x, y, live)
	if r.capture != nil {
		pose := r.defs.Objects[r.capture.ID].Capture
		x, y = pose.CenterX, pose.CenterY
	}
	if rollovers {
		r.sensorList(layer.ExtraSensors, x, y, live)
	} else {
		for _, sensor := range layer.ExtraSensors {
			r.hits[sensor.Object] = false
		}
	}
}

func (r *Engine) sensorList(sensors []ruledata.Sensor, x, y int, live bool) {
	stopped := false
	for _, sensor := range sensors {
		// The ordered scan tests the lower Y bound before the upper X bound.
		// Once it reaches a later vertical region, all remaining latches clear.
		if !stopped && x >= sensor.X0 && y < sensor.Y0 {
			stopped = true
			if r.defs.KeepLatchAtYStop {
				// Later tables advance past this record before clearing later
				// latches. Its previous entry state therefore survives the stop.
				continue
			}
		}
		inside := !stopped && x >= sensor.X0 && x <= sensor.X1 && y >= sensor.Y0 && y <= sensor.Y1
		if !inside {
			r.hits[sensor.Object] = false
			continue
		}
		stopped = true
		if r.hits[sensor.Object] {
			continue
		}
		r.hits[sensor.Object] = true
		object := r.defs.Objects[sensor.Object]
		role := object.Role
		if role != ruledata.TrapObject {
			r.captureContact = false
		}
		if role == ruledata.AcceleratorObject || role == ruledata.TrapObject {
			r.placeCapture(sensor.Object, live)
		} else if live {
			r.Hit(sensor.Object, role == ruledata.GateEnterObject)
		} else {
			if role == ruledata.GateEnterObject {
				r.feature(object.Event)
			}
			r.applyObjectGate(object)
		}
	}
}

// Message returns an original bonus/feature announcement while it is active.
func (r *Engine) Message() string { return r.message }

// BoosterBonus returns the source booster contribution to the retained bonus bank.
func (r *Engine) BoosterBonus() uint64 {
	return uint64(r.actorFeatures().Boosters) * r.defs.BoosterValue
}
