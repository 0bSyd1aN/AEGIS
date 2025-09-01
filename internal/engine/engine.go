package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// Prometheus counter
var processedCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "packets_processed_total",
		Help: "Total number of packets processed",
	},
)

func init() {
	prometheus.MustRegister(processedCounter)
}

// Engine struct
type Engine struct {
	workers   int
	queue     chan []byte
	stopped   chan struct{}
	logger    zerolog.Logger
	wg        sync.WaitGroup
	processed uint64
	stopOnce  sync.Once
}

// Constructor
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

// Start engine workers
func (e *Engine) Start(ctx context.Context) error {
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx)
	}
	e.logger.Info().Int("workers", e.workers).Msg("engine started")
	return nil
}

// Worker goroutine
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

// Process a packet
func (e *Engine) process(pkt []byte) {
	time.Sleep(5 * time.Millisecond)
	atomic.AddUint64(&e.processed, 1)
	processedCounter.Inc()
	e.logger.Debug().Int("len", len(pkt)).Msg("packet processed")
}

// Enqueue packet
func (e *Engine) Enqueue(pkt []byte) {
	select {
	case e.queue <- pkt:
	default:
		e.logger.Warn().Msg("queue full, dropping packet")
	}
}

// Stop engine
func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopped)
		e.logger.Info().Msg("engine stopping: waiting for workers")
		e.wg.Wait()
		e.logger.Info().Msg("engine stopped")
	})
}
