// Package physics is a native translation of the original integer ball model.
// No CPU, custom-chip registers or hardware interpreter are present at runtime.
package physics

import (
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

const (
	TicksPerSecond  = 50
	SubstepsPerTick = 4
	PositionShift   = 10
	angleCount      = 2048
	velocityLimit   = 4000
)

// State uses the original units: positions have ten fractional bits and
// velocities are position increments per one of four simulation substeps.
type State struct {
	X, Y                                               int32
	VX, VY, Spin                                       int16
	LeftSpeed, RightSpeed, LeftPosition, RightPosition int16
	NudgePosition, NudgeSpeed                          int16
	LeftPhase, RightPhase                              int16
}

// Controls is sampled once per 50 Hz frame and shared by the four substeps.
type Controls struct{ Left, Right, Nudge bool }

// Contact describes the source perimeter result and moving surface velocity.
type Contact struct {
	PointX, PointY         int
	Angle, Depth, Material int
	SurfaceX, SurfaceY     int16
	Powered                bool
	Object                 int
}

// Engine owns deterministic gameplay state and immutable original geometry.
type Engine struct {
	Layer              int
	dynamicLayers      [2][]byte
	Locked             bool
	DynamicCollision   []byte
	FrameContact       Contact
	HadContact         bool
	ImpactObject       int
	Table              *resources.Table
	State              State
	Controls           Controls
	LastContact        Contact
	Tilted             bool
	GravityX, GravityY int16
	gravitySelected    bool
	visual             visualPose
	// MovingFlipperContact retains early substep contacts even when a later
	// stationary contact replaces FrameContact. It does not change response.
	MovingFlipperContact bool
}

func New(table *resources.Table) *Engine {
	return &Engine{Table: table, State: State{X: 302 << 10, Y: 470 << 10, LeftPosition: 55, RightPosition: 55}, GravityX: table.GravityDefault[0], GravityY: table.GravityDefault[1]}
}

// ResetBall resets motion while retaining the table's current target geometry.
// The new-ball restoration service then raises each movable target in order.
func (e *Engine) ResetBall() {
	e.SetLayer(0)
	fresh := New(e.Table)
	fresh.DynamicCollision, fresh.dynamicLayers = e.DynamicCollision, e.dynamicLayers
	*e = *fresh
}

// SelectLayer switches all collision references through the original ramp gates.
func (e *Engine) SelectLayer() {
	e.SetLayer(e.Table.SelectLayer(e.Layer, e.State.PixelX()+8, e.State.PixelY()+8))
}

// SetLayer retains each level's mutable target raster while the ball is away.
func (e *Engine) SetLayer(level int) {
	if level == e.Layer {
		return
	}
	e.dynamicLayers[e.Layer] = e.DynamicCollision
	e.Layer = level
	e.DynamicCollision = e.dynamicLayers[level]
}

func (e *Engine) collision() []byte {
	if e.DynamicCollision != nil {
		return e.DynamicCollision
	}
	return e.Table.Layers[e.Layer].Collision
}

func (s *State) PixelX() int { return int(s.X >> PositionShift) }
func (s *State) PixelY() int { return int(s.Y >> PositionShift) }
func (s *State) Nudge() int  { return int(s.NudgePosition >> PositionShift) }
func (s *State) LeftFrame() int {
	return int(s.LeftPhase)
}
func (s *State) RightFrame() int {
	return int(s.RightPhase)
}

// Step advances contacts, response, flippers and integration. The caller then
// samples the next frame's force before applying sensor and ramp-level changes.
func (e *Engine) Step(input Controls) {
	e.CaptureVisualNudge()
	e.Controls = input
	e.HadContact = false
	e.MovingFlipperContact = false
	e.ImpactObject = 0
	var armed Contact
	if !e.Locked {
		if !e.gravitySelected {
			e.GravityX, e.GravityY = e.Table.GravityDefault[0], e.Table.GravityDefault[1]
		}
		// A selected field survives one unlocked frame. The live sampler
		// may re-arm it afterward; restricted loops have no such service.
		e.gravitySelected = false
	}
	for substep := range SubstepsPerTick {
		e.LastContact = Contact{}
		if !e.Locked {
			if contact, ok := e.Probe(); ok {
				if (contact.Material == 1 || contact.Material == 2) && (contact.SurfaceX != 0 || contact.SurfaceY != 0) {
					e.MovingFlipperContact = true
				}
				// A qualified powered contact remains armed between substeps
				// until a response rejects it or a later one replaces it.
				if !contact.Powered && armed.Object != 0 {
					contact.Powered, contact.Object = true, armed.Object
				}
				contact.Powered = e.Resolve(contact)
				if contact.Powered {
					armed = contact
				} else {
					armed = Contact{}
					contact.Object = 0
				}
				e.FrameContact, e.HadContact = contact, true
				e.LastContact = contact
			}
		}
		e.Integrate()
		if substep == 0 && !e.Locked {
			e.recordVisualPose()
		}
	}
	if e.Locked {
		e.recordVisualPose()
	}
	// Gameplay consumes only the impact still armed at the end of the frame.
	// An early rebound can be canceled by a later substep in the same frame.
	e.ImpactObject = armed.Object
}
