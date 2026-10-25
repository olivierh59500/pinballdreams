package rules

import "github.com/olivierh59500/pinballdreams/internal/ruledata"

type capture struct {
	ID, Event, Period, Remaining, Counter, QueueBase int
	Role                                             ruledata.ObjectRole
}

// captureBall places the ball at a decoded trap or accelerator pose and queues
// its feature. Stored velocity is retained while its presentation runs.
func (r *Engine) captureBall(id int) {
	r.placeCapture(id, true)
}

func (r *Engine) placeCapture(id int, live bool) {
	object := r.defs.Objects[id]
	if object.Role == ruledata.TrapObject && r.enterLock(id) {
		r.captureContact = false
		r.capture = nil
		r.Locked = false
		r.capturePose(id)
		return
	}
	pose := object.Capture
	r.captureContact = object.Role == ruledata.TrapObject
	r.capture = &capture{ID: id, Role: object.Role, Event: object.Event, Period: pose.Period, Remaining: pose.Count, Counter: 1, QueueBase: len(r.queue)}
	r.Locked = true
	r.feature(object.Event)
	if live {
		r.setLamp(object.Lamp, true)
	}
	r.capturePose(id)
	if live && object.Role == ruledata.AcceleratorObject && pose.Period == 0 {
		r.capture = nil
		r.Locked = false
	}
}

func (r *Engine) capturePose(id int) {
	pose := r.defs.Objects[id].Capture
	if r.OnCapture != nil {
		r.OnCapture(pose.CenterX-8, pose.CenterY-8, pose.VX, pose.VY)
	}
}

// Accelerator advances at the two original per-frame score-bank service points.
func (r *Engine) Accelerator() {
	c := r.capture
	if c == nil || c.Role != ruledata.AcceleratorObject {
		return
	}
	c.Counter--
	if c.Counter != 0 {
		return
	}
	c.Counter = c.Period
	c.Remaining--
	if c.Remaining == 0 {
		r.capture = nil
		r.Locked = false
		return
	}
	award := r.defs.Events[c.Event]
	r.award(award.Points, award.Bonus)
}
