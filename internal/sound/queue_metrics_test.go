package sound

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func assertQueueConservation(t *testing.T, metrics PCMQueueMetrics) {
	t.Helper()
	available := metrics.InitialBufferedBytes + metrics.ProducedBytes + metrics.InsertedReserveBytes
	accounted := metrics.SourceReadBytes + metrics.DiscardedBytes + uint64(metrics.BufferedBytes)
	if available != accounted {
		t.Fatalf("queue byte conservation: available=%d accounted=%d metrics=%+v", available, accounted, metrics)
	}
}

func TestQueueMetricsKeepSourcePrefetchAndCancelledReserveDistinct(t *testing.T) {
	queue := NewPCMQueue(2)
	queue.SetMetricsEnabled(true)
	queue.Push(make([]byte, 12))
	for _, request := range []int{1, FrameBytes * 2} {
		if _, err := queue.Read(make([]byte, request)); err != nil {
			t.Fatal(err)
		}
	}
	discarded := queue.BufferedBytes()
	queue.Reset(1)
	queue.Read(nil)
	queue.Read(make([]byte, 5000))
	queue.Read(make([]byte, 5000))
	metrics := queue.Metrics()
	assertQueueConservation(t, metrics)
	if !metrics.Enabled || metrics.InitialBufferedBytes != 2*FrameBytes || metrics.ProducedBytes != 12 || metrics.SourceReadBytes != 2*FrameBytes+1 {
		t.Fatalf("source accounting differs: %+v", metrics)
	}
	if metrics.DiscardedBytes != uint64(discarded) || metrics.InsertedReserveBytes != FrameBytes || metrics.ResetCalls != 1 {
		t.Fatalf("reset counted cancellation as playback: %+v", metrics)
	}
	if metrics.ReadCalls != 5 || metrics.EmptyReads != 1 || metrics.ZeroLengthReads != 1 || metrics.MinimumRequestBytes != 1 || metrics.MaximumRequestBytes != FrameBytes*2 || metrics.LastRequestBytes != 5000 {
		t.Fatalf("software read request accounting differs: %+v", metrics)
	}
	if metrics.MostCommonRecordedRequestBytes != 5000 || len(metrics.ReadRequests) != 4 || metrics.BufferedBytes != 0 || metrics.BufferedDuration != 0 || metrics.MinimumBufferedBytes != 0 || metrics.MaximumBufferedBytes != 2*FrameBytes+12 {
		t.Fatalf("buffer/request snapshot differs: %+v", metrics)
	}
}

func TestQueueMetricsEnableBoundaryAndSnapshotsAreIndependent(t *testing.T) {
	queue := NewPCMQueue(1)
	if metrics := queue.Metrics(); metrics.Enabled || metrics.BufferedDuration != 20*time.Millisecond || len(metrics.ReadRequests) != 0 {
		t.Fatalf("disabled diagnostics changed initial queue semantics: %+v", metrics)
	}
	queue.Read(make([]byte, FrameBytes))
	queue.Push([]byte{1, 2, 3, 4})
	queue.SetMetricsEnabled(true)
	queue.Read(make([]byte, 1))
	queue.SetMetricsEnabled(true)
	metrics := queue.Metrics()
	if metrics.InitialBufferedBytes != 4 || metrics.ProducedBytes != 0 || metrics.SourceReadBytes != 1 || metrics.ReadCalls != 1 {
		t.Fatalf("enabling did not retain its observation boundary: %+v", metrics)
	}
	metrics.ReadRequests[0].Bytes = 999
	if queue.Metrics().ReadRequests[0].Bytes != 1 {
		t.Fatal("a caller mutated live accounting through a snapshot")
	}
	queue.SetMetricsEnabled(false)
	queue.Read(make([]byte, 1))
	if metrics := queue.Metrics(); metrics.Enabled || metrics.SourceReadBytes != 0 || metrics.BufferedBytes != 2 {
		t.Fatalf("disabled accounting retained counters or changed bytes: %+v", metrics)
	}
	queue.SetMetricsEnabled(true)
	metrics = queue.Metrics()
	assertQueueConservation(t, metrics)
	if metrics.InitialBufferedBytes != 2 || metrics.ReadCalls != 0 {
		t.Fatal("re-enabling did not start a fresh observation period")
	}
}

func TestQueueReadRequestDoesNotRepresentDeviceBurst(t *testing.T) {
	queue := NewPCMQueue(7)
	queue.SetMetricsEnabled(true)
	// Oto's 100 ms source-buffer request can consume only our 20 ms source
	// read cap. No device callback has executed in this test.
	request := make([]byte, SampleRate*100/1000*4)
	count, err := queue.Read(request)
	if err != nil || count != FrameBytes {
		t.Fatalf("source prefetch read: %d %v", count, err)
	}
	metrics := queue.Metrics()
	if metrics.MostCommonRecordedRequestBytes != len(request) || metrics.SourceReadBytes != FrameBytes || metrics.BufferedDuration != 120*time.Millisecond {
		t.Fatalf("source request/cap accounting differs: %+v", metrics)
	}
	assertQueueConservation(t, metrics)
}

func TestQueueRequestHistogramIsBoundedAndDisclosesOverflow(t *testing.T) {
	queue := NewPCMQueue(0)
	queue.SetMetricsEnabled(true)
	request := make([]byte, 20)
	for size := 1; size <= len(request); size++ {
		queue.Read(request[:size])
	}
	for range 3 {
		queue.Read(request[:7])
	}
	metrics := queue.Metrics()
	if len(metrics.ReadRequests) != 16 || metrics.OtherRequestCalls != 4 || metrics.ReadCalls != 23 || metrics.EmptyReads != 23 || metrics.MostCommonRecordedRequestBytes != 7 {
		t.Fatalf("bounded histogram lost its overflow/mode accounting: %+v", metrics)
	}
	assertQueueConservation(t, metrics)
}

func TestQueueAccountingAddsNoProductionReadPathAllocations(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		queue := NewPCMQueue(0)
		queue.SetMetricsEnabled(enabled)
		block, output := make([]byte, FrameBytes), make([]byte, FrameBytes)
		allocations := testing.AllocsPerRun(100, func() {
			queue.Push(block)
			queue.Read(output)
		})
		if allocations != 0 {
			t.Fatalf("enabled=%t: push/read allocated %g times", enabled, allocations)
		}
	}
}

func TestQueueAccountingConcurrentProducerReaderAndInspection(t *testing.T) {
	queue := NewPCMQueue(0)
	queue.SetMetricsEnabled(true)
	const blocks = 500
	var producer sync.WaitGroup
	producer.Add(1)
	go func() {
		defer producer.Done()
		block := make([]byte, FrameBytes)
		for range blocks {
			queue.Push(block)
		}
	}()
	done := make(chan struct{})
	go func() { producer.Wait(); close(done) }()
	output := make([]byte, 5000)
	deadline := time.Now().Add(5 * time.Second)
	for calls := 0; ; calls++ {
		queue.Read(output)
		if calls%20 == 0 {
			assertQueueConservation(t, queue.Metrics())
		}
		select {
		case <-done:
			if queue.BufferedBytes() == 0 {
				metrics := queue.Metrics()
				assertQueueConservation(t, metrics)
				if metrics.ProducedBytes != blocks*FrameBytes || metrics.SourceReadBytes != blocks*FrameBytes {
					t.Fatalf("concurrent accounting lost produced/read bytes: %+v", metrics)
				}
				return
			}
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("concurrent source queue did not drain")
		}
		runtime.Gosched()
	}
}
