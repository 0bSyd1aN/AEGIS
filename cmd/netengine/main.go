package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/your-org/aegis-netengine/internal/engine"
)

func main() {
	listenAddr := flag.String("listen", ":9000", "UDP listen address")
	httpAddr := flag.String("http", ":9090", "HTTP metrics address")
	workers := flag.Int("workers", 4, "number of workers")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.With().Str("component", "netengine").Logger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	eng := engine.New(*workers, logger)
	if err := eng.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("engine start failed")
	}

	// Start simple HTTP server for metrics & health
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: *httpAddr, Handler: mux}
	go func() {
		logger.Info().Str("http", *httpAddr).Msg("http server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("http server error")
		}
	}()

	// UDP listener
	go func() {
		conn, err := net.ListenPacket("udp", *listenAddr)
		if err != nil {
			logger.Fatal().Err(err).Msg("udp listen failed")
		}
		defer conn.Close()
		buf := make([]byte, 65535)
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Error().Err(err).Msg("udp read")
				continue
			}
			// copy packet to avoid reuse issue
			b := make([]byte, n)
			copy(b, buf[:n])
			logger.Debug().Str("peer", addr.String()).Int("size", n).Msg("packet received")
			eng.Enqueue(b)
		}
	}()

	<-ctx.Done()
	logger.Info().Msg("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	eng.Stop()
}
