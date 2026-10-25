// Package platforminput preserves platform input transitions until sampled.
package platforminput

import "sync"

type TouchAction uint8

const (
	TouchDown TouchAction = iota
	TouchUp
	TouchMove
)

// Touch coordinates use the platform adapter's device-independent units.
type Touch struct {
	Action TouchAction
	ID     int
	X, Y   float64
}

type touchPhase uint8

const (
	touchIdle touchPhase = iota
	touchPressSent
	touchPressObserved
	touchReleaseSent
)

type touchState struct {
	phase   touchPhase
	pending []Touch
}

// TouchQueue keeps a press and its release on distinct input snapshots. A
// normal press/move is forwarded immediately; only an unobserved release or
// a reused pointer's next press needs to wait. The callback must not reenter
// this queue. Android callbacks and the game-thread acknowledgement may race.
type TouchQueue struct {
	mu       sync.Mutex
	pointers map[int]*touchState
	dispatch func(Touch)
}

func NewTouchQueue(dispatch func(Touch)) *TouchQueue {
	return &TouchQueue{pointers: make(map[int]*touchState), dispatch: dispatch}
}

func (q *TouchQueue) Push(event Touch) {
	q.mu.Lock()
	defer q.mu.Unlock()
	pointer := q.pointers[event.ID]
	if pointer == nil {
		if event.Action != TouchDown {
			return
		}
		pointer = &touchState{}
		q.pointers[event.ID] = pointer
	}
	if len(pointer.pending) == 0 && event.Action == TouchMove && (pointer.phase == touchPressSent || pointer.phase == touchPressObserved) {
		q.dispatch(event)
		return
	}
	// Motion can coalesce, but press and release boundaries retain their order.
	if event.Action == TouchMove && len(pointer.pending) > 0 && pointer.pending[len(pointer.pending)-1].Action == TouchMove {
		pointer.pending[len(pointer.pending)-1] = event
	} else {
		pointer.pending = append(pointer.pending, event)
	}
	q.flush(pointer)
}

func (q *TouchQueue) flush(pointer *touchState) {
	for len(pointer.pending) > 0 {
		event := pointer.pending[0]
		switch pointer.phase {
		case touchIdle:
			if event.Action == TouchDown {
				q.dispatch(event)
				pointer.phase = touchPressSent
			}
		case touchPressSent:
			if event.Action != TouchMove {
				return
			}
			q.dispatch(event)
		case touchPressObserved:
			if event.Action == TouchDown {
				return
			}
			q.dispatch(event)
			if event.Action == TouchUp {
				pointer.phase = touchReleaseSent
			}
		case touchReleaseSent:
			return
		}
		pointer.pending = pointer.pending[1:]
	}
}

// Observe acknowledges the touch IDs sampled by the just-completed game poll.
// A reused ID cannot start a new gesture before a neutral snapshot is seen.
func (q *TouchQueue) Observe(ids []int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for id, pointer := range q.pointers {
		down := false
		for _, observed := range ids {
			down = down || observed == id
		}
		if pointer.phase == touchPressSent && down {
			pointer.phase = touchPressObserved
		} else if pointer.phase == touchReleaseSent && !down {
			pointer.phase = touchIdle
		}
		q.flush(pointer)
		if pointer.phase == touchIdle && len(pointer.pending) == 0 {
			delete(q.pointers, id)
		}
	}
}

// Cancel retires pending gestures immediately. A following press still waits
// for the cancelled ID to disappear from a sampled input frame.
func (q *TouchQueue) Cancel(id int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if pointer := q.pointers[id]; pointer != nil {
		pointer.pending = nil
		q.dispatch(Touch{Action: TouchUp, ID: id})
		pointer.phase = touchReleaseSent
	}
}

func (q *TouchQueue) CancelAll() {
	q.mu.Lock()
	defer q.mu.Unlock()
	for id, pointer := range q.pointers {
		pointer.pending = nil
		q.dispatch(Touch{Action: TouchUp, ID: id})
		pointer.phase = touchReleaseSent
	}
}
