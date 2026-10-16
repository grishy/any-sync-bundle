package cmd

import (
	"context"
	"errors"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func startPprofServer(ctx context.Context, c *cli.Context) {
	if !c.Bool(flagPprof) {
		return
	}

	addr := c.String(flagPprofAddr)
	log.Info("🔍 starting pprof HTTP server",
		zap.String("addr", addr),
		zap.String("url", "http://"+addr+"/debug/pprof/"))

	// A private mux avoids exposing pprof through the process-wide default mux.
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("pprof server failed", zap.Error(err))
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Warn("pprof server shutdown failed", zap.Error(err))
		}
	}()

	log.Info("✓ pprof server started - use 'go tool pprof' to analyze",
		zap.String("cpu_profile", "go tool pprof http://"+addr+"/debug/pprof/profile?seconds=30"),
		zap.String("heap_profile", "go tool pprof http://"+addr+"/debug/pprof/heap"),
		zap.String("goroutine_profile", "go tool pprof http://"+addr+"/debug/pprof/goroutine"))
}
