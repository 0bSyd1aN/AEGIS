package engine

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

func New(workers int, logger zerolog.Logger) *Engine {
	if workers <= 0 {
		workers = 4
	}
	return &Engine{
		workers: workers,
		queue:   make(chan []byte, 10000),
		stopped: make(chan struct{}),
		logger:  logger.With().Str("component", "engine").Logger(),
	}
}

func (e *Engine) Start(ctx context.Context) error {
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx)
	}
	e.logger.Info().Int("workers", e.workers).Msg("engine started")
	return nil
}

func (e *Engine) worker(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case pkt := <-e.queue:
			e.process(pkt)
		}
	}
}

func (e *Engine) process(pkt []byte) {
	// Simulate work: replace with real parsing/inspect/forwarding
	time.Sleep(5 * time.Millisecond)
	atomic.AddUint64(&e.processed, 1)
	processedCounter.Inc()
	e.logger.Debug().Int("len", len(pkt)).Msg("packet processed")
}

// Enqueue puts a packet into the processing queue (non-blocking)
func (e *Engine) Enqueue(pkt []byte) {
	select {
	case e.queue <- pkt:
	default:
		// backpressure/drop: real product should have metrics and configurable policy
		e.logger.Warn().Msg("queue full, dropping packet")
	}
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopped)
		e.logger.Info().Msg("engine stopping: waiting for workers")
		e.wg.Wait()
		e.logger.Info().Msg("engine stopped")
	})
}
