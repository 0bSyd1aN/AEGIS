package engine

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func Test_engine_process(t *testing.T) {
	logger := zerolog.Nop()
	eng := New(2, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start engine workers
	eng.Start(ctx)

	var counter int32

	// Submit jobs as packets (simulate jobs as byte slices)
	for i := 0; i < 5; i++ {
		jobID := i
		data := []byte{byte(i)} // dummy packet
		eng.Enqueue(data)
		t.Logf("job %d submitted", jobID)
		atomic.AddInt32(&counter, 1)
	}

	// Wait a bit for jobs to finish
	time.Sleep(500 * time.Millisecond)

	// Check that all jobs ran
	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("expected 5 jobs, got %d", counter)
	}
}
