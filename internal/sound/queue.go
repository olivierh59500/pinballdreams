package sound

import (
	"sync"
	"time"
)

// PCMReadRequest groups requests made by the queue's software reader. Returned
// bytes may be prefetched into a player; they are not device playback evidence.
// The pinned Oto mux requests its player source-buffer size, not the device's
// callback size or current source-buffer deficiency. Do not infer driver bursts
// or physical consumption from this histogram.
type PCMReadRequest struct {
	Bytes       int
	Calls       uint64
	SourceBytes uint64
	EmptyCalls  uint64
}

// PCMQueueMetrics accounts only for this queue and its immediate reader.
// BufferedDuration excludes player/device buffers and is not audible latency.
// No field reports actual device consumption, callback size or driver latency.
type PCMQueueMetrics struct {
	Enabled                                                    bool
	InitialBufferedBytes, ProducedBytes, SourceReadBytes       uint64
	ReadCalls, EmptyReads, ZeroLengthReads, ResetCalls         uint64
	DiscardedBytes, InsertedReserveBytes                       uint64
	BufferedBytes, MinimumBufferedBytes, MaximumBufferedBytes  int
	BufferedDuration                                           time.Duration
	MinimumRequestBytes, MaximumRequestBytes, LastRequestBytes int
	// The mode covers recorded positive request sizes. OtherRequestCalls
	// discloses calls whose size exceeded the fixed distinct-size budget.
	MostCommonRecordedRequestBytes int
	OtherRequestCalls              uint64
	ReadRequests                   []PCMReadRequest
}

type pcmQueueAccounting struct {
	values   PCMQueueMetrics
	requests [16]PCMReadRequest
	kinds    int
}

// PCMQueue stores already rendered audio in order. Its consumer never calls
// a tracker or game routine. Empty reads are temporary, not end of stream.
type PCMQueue struct {
	mu           sync.Mutex
	data         []byte
	start, count int
	metrics      *pcmQueueAccounting
}

// NewPCMQueue reserves complete silent frames for backend callback jitter.
// The reserve delays output; it does not advance any musical state.
func NewPCMQueue(reserveFrames int) *PCMQueue {
	if reserveFrames < 0 {
		panic("sound: negative PCM reserve")
	}
	return &PCMQueue{data: make([]byte, max(8, reserveFrames+1)*FrameBytes), count: reserveFrames * FrameBytes}
}

// Push copies one or more produced blocks before their reusable canvas changes.
// Growth preserves all queued samples instead of dropping game sound events.
func (q *PCMQueue) Push(pcm []byte) {
	if len(pcm) == 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.data)-q.count < len(pcm) {
		data := make([]byte, max(len(q.data)*2, q.count+len(pcm)))
		first := min(q.count, len(q.data)-q.start)
		copy(data, q.data[q.start:q.start+first])
		copy(data[first:], q.data[:q.count-first])
		q.data, q.start = data, 0
	}
	at := (q.start + q.count) % len(q.data)
	first := min(len(pcm), len(q.data)-at)
	copy(q.data[at:], pcm[:first])
	copy(q.data, pcm[first:])
	q.count += len(pcm)
	if accounting := q.metrics; accounting != nil {
		accounting.values.ProducedBytes += uint64(len(pcm))
		accounting.watermark(q.count)
	}
}

// Read accepts short and partial stereo reads. Each call is bounded to one
// game frame, so a refill cannot append another full backend buffer at once.
func (q *PCMQueue) Read(out []byte) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := min(len(out), q.count, FrameBytes)
	if accounting := q.metrics; accounting != nil {
		accounting.read(len(out), n, q.count-n)
	}
	if n == 0 {
		return 0, nil
	}
	first := min(n, len(q.data)-q.start)
	copy(out, q.data[q.start:q.start+first])
	copy(out[first:n], q.data[:n-first])
	q.start = (q.start + n) % len(q.data)
	q.count -= n
	return n, nil
}

func (q *PCMQueue) BufferedBytes() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.count
}

// Reset retires cancelled output and restores the requested silent reserve.
// The caller must stop its backend reader first so prefetched data can also be
// discarded. Ordinary suspension keeps the queue and does not call Reset.
func (q *PCMQueue) Reset(reserveFrames int) {
	if reserveFrames < 0 {
		panic("sound: negative PCM reserve")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	count := reserveFrames * FrameBytes
	if accounting := q.metrics; accounting != nil {
		accounting.values.ResetCalls++
		accounting.values.DiscardedBytes += uint64(q.count)
		accounting.values.InsertedReserveBytes += uint64(count)
		accounting.watermark(count)
	}
	if len(q.data) < count {
		q.data = make([]byte, max(8, reserveFrames+1)*FrameBytes)
	}
	clear(q.data[:count])
	q.start, q.count = 0, count
}

// SetMetricsEnabled starts optional accounting at the current queue contents.
// Repeated enabling retains the measurement; disabling discards it. Ordinary
// playback leaves accounting nil and adds no read-path timers or allocations.
func (q *PCMQueue) SetMetricsEnabled(enabled bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !enabled {
		q.metrics = nil
	} else if q.metrics == nil {
		q.metrics = &pcmQueueAccounting{values: PCMQueueMetrics{
			Enabled: true, InitialBufferedBytes: uint64(q.count),
			MinimumBufferedBytes: q.count, MaximumBufferedBytes: q.count,
		}}
	}
}

// Metrics returns an independent snapshot. SourceReadBytes counts bytes removed
// by the immediate reader, including silence and prefetch. EmptyReads describes
// an empty software queue, not a proven device underrun. Snapshot allocation is
// outside the render/read path; the caller may poll at a low diagnostic rate.
func (q *PCMQueue) Metrics() PCMQueueMetrics {
	q.mu.Lock()
	defer q.mu.Unlock()
	var result PCMQueueMetrics
	if accounting := q.metrics; accounting != nil {
		result = accounting.values
		result.ReadRequests = append([]PCMReadRequest(nil), accounting.requests[:accounting.kinds]...)
		var modeCalls uint64
		for _, request := range result.ReadRequests {
			if request.Bytes > 0 && request.Calls > modeCalls {
				result.MostCommonRecordedRequestBytes, modeCalls = request.Bytes, request.Calls
			}
		}
	}
	result.BufferedBytes = q.count
	result.BufferedDuration = time.Duration(q.count) * time.Second / (SampleRate * 4)
	return result
}

func (a *pcmQueueAccounting) watermark(buffered int) {
	a.values.MinimumBufferedBytes = min(a.values.MinimumBufferedBytes, buffered)
	a.values.MaximumBufferedBytes = max(a.values.MaximumBufferedBytes, buffered)
}

func (a *pcmQueueAccounting) read(requested, returned, buffered int) {
	v := &a.values
	v.ReadCalls++
	v.SourceReadBytes += uint64(returned)
	v.LastRequestBytes = requested
	v.MaximumRequestBytes = max(v.MaximumRequestBytes, requested)
	if requested == 0 {
		v.ZeroLengthReads++
	} else {
		if v.MinimumRequestBytes == 0 || requested < v.MinimumRequestBytes {
			v.MinimumRequestBytes = requested
		}
		if returned == 0 {
			v.EmptyReads++
		}
	}
	a.watermark(buffered)
	index := 0
	for index < a.kinds && a.requests[index].Bytes != requested {
		index++
	}
	if index == len(a.requests) {
		v.OtherRequestCalls++
		return
	}
	if index == a.kinds {
		a.requests[index].Bytes = requested
		a.kinds++
	}
	request := &a.requests[index]
	request.Calls++
	request.SourceBytes += uint64(returned)
	if requested > 0 && returned == 0 {
		request.EmptyCalls++
	}
}
