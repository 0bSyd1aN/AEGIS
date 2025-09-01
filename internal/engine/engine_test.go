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
	eng := New(2, logger) // assuming New(workers, logger) returns *Engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var counter int32

	// Submit some jobs
	for i := 0; i < 5; i++ {
		jobID := i
		eng.Submit(func() {
			atomic.AddInt32(&counter, 1)
			t.Logf("job %d executed", jobID)
		})
	}

	// Run the engine
	go eng.Process(ctx)

	// Wait a bit for jobs to finish
	time.Sleep(500 * time.Millisecond)

	// Check that all jobs ran
	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("expected 5 jobs, got %d", counter)
	}
}
