package platforminput

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

type touchHarness struct {
	sync.Mutex
	down   map[int]Touch
	events []Touch
}

func newTouchHarness() (*TouchQueue, *touchHarness) {
	h := &touchHarness{down: make(map[int]Touch)}
	return NewTouchQueue(func(event Touch) {
		h.Lock()
		defer h.Unlock()
		h.events = append(h.events, event)
		if event.Action == TouchUp {
			delete(h.down, event.ID)
		} else {
			h.down[event.ID] = event
		}
	}), h
}

func (h *touchHarness) snapshot() []int {
	h.Lock()
	defer h.Unlock()
	var ids []int
	for id := range h.down {
		ids = append(ids, id)
	}
	return ids
}

func TestTapBetweenSnapshotsIsObservedBeforeRelease(t *testing.T) {
	queue, h := newTouchHarness()
	queue.Push(Touch{Action: TouchDown, ID: 7, X: 232, Y: 10})
	queue.Push(Touch{Action: TouchUp, ID: 7, X: 232, Y: 10})
	ids := h.snapshot()
	if len(ids) != 1 || ids[0] != 7 {
		t.Fatal("a complete tap disappeared before its first input snapshot")
	}
	queue.Observe(ids)
	if len(h.snapshot()) != 0 {
		t.Fatal("the acknowledged tap did not release")
	}
	queue.Observe(nil)
	if len(h.events) != 2 || h.events[0].Action != TouchDown || h.events[1].Action != TouchUp {
		t.Fatal("tap transitions were duplicated or reordered")
	}
}

func TestReusedPointerGetsNeutralFrameBetweenTaps(t *testing.T) {
	queue, h := newTouchHarness()
	for _, x := range []float64{10, 200} {
		queue.Push(Touch{Action: TouchDown, ID: 0, X: x})
		queue.Push(Touch{Action: TouchUp, ID: 0, X: x})
	}
	for poll, want := range []int{1, 0, 1, 0} {
		ids := h.snapshot()
		if len(ids) != want {
			t.Fatalf("poll %d: expected %d contacts, got %v", poll, want, ids)
		}
		queue.Observe(ids)
	}
	if len(h.events) != 4 || h.events[0].X != 10 || h.events[2].X != 200 {
		t.Fatal("a reused pointer merged two taps or changed their positions")
	}
}

func TestMotionAndOtherFingersRemainImmediate(t *testing.T) {
	queue, h := newTouchHarness()
	queue.Push(Touch{Action: TouchDown, ID: 0})
	queue.Push(Touch{Action: TouchMove, ID: 0, X: 30, Y: 40})
	if h.down[0].X != 30 || h.down[0].Y != 40 {
		t.Fatal("input acknowledgement delayed ordinary motion")
	}
	queue.Observe(h.snapshot())
	queue.Push(Touch{Action: TouchDown, ID: 1})
	queue.Push(Touch{Action: TouchUp, ID: 1})
	queue.Observe(h.snapshot())
	if ids := h.snapshot(); len(ids) != 1 || ids[0] != 0 {
		t.Fatal("a second finger's release changed the held first finger")
	}
	queue.Push(Touch{Action: TouchUp, ID: 0})
	if len(h.snapshot()) != 0 {
		t.Fatal("an already observed finger's ordinary release was delayed")
	}
}

func TestCancellationDropsOldTapAndPreservesFreshGesture(t *testing.T) {
	queue, h := newTouchHarness()
	queue.Push(Touch{Action: TouchDown, ID: 0, X: 20})
	queue.Push(Touch{Action: TouchUp, ID: 0, X: 20})
	queue.Cancel(0)
	queue.Push(Touch{Action: TouchDown, ID: 0, X: 240})
	if len(h.snapshot()) != 0 {
		t.Fatal("fresh gesture crossed the cancelled contact's neutral boundary")
	}
	queue.Observe(nil)
	if ids := h.snapshot(); len(ids) != 1 || h.down[0].X != 240 {
		t.Fatal("cancellation absorbed the next fresh gesture")
	}
	queue.CancelAll()
	queue.Observe(nil)
	if len(h.snapshot()) != 0 || len(queue.pointers) != 0 {
		t.Fatal("suspension retained a touch or pending tap")
	}
}

func TestConcurrentPlatformPacketsAndGameSnapshots(t *testing.T) {
	queue, h := newTouchHarness()
	var producers sync.WaitGroup
	for id := range 2 {
		producers.Add(1)
		go func() {
			defer producers.Done()
			for range 30 {
				queue.Push(Touch{Action: TouchDown, ID: id})
				queue.Push(Touch{Action: TouchUp, ID: id})
			}
		}()
	}
	done := make(chan struct{})
	go func() { producers.Wait(); close(done) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		queue.Observe(h.snapshot())
		select {
		case <-done:
			queue.mu.Lock()
			remaining := len(queue.pointers)
			queue.mu.Unlock()
			if remaining == 0 {
				h.Lock()
				count := len(h.events)
				h.Unlock()
				if count != 120 {
					t.Fatalf("concurrent delivery retained %d of 120 transitions", count)
				}
				return
			}
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("touch transitions did not drain")
		}
		runtime.Gosched()
	}
}
