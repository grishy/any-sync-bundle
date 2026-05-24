package doctor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	doctorRunPath           = "/doctor/run"
	socketMode              = 0o600
	doctorReadHeaderTimeout = 5 * time.Second
)

type ServerConfig struct {
	SocketPath string
	Runner     Runner
}

type Server struct {
	cfg ServerConfig

	httpServer *http.Server
	closeOnce  sync.Once
	closeErr   error

	mu      sync.Mutex
	running bool
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.cfg.Runner == nil {
		return errors.New("doctor runner is required")
	}
	if s.cfg.SocketPath == "" {
		return errors.New("doctor socket path is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.cfg.SocketPath), 0o750); err != nil {
		return fmt.Errorf("create doctor socket directory: %w", err)
	}
	if err := removeStaleSocket(s.cfg.SocketPath); err != nil {
		return err
	}

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "unix", s.cfg.SocketPath)
	if err != nil {
		return fmt.Errorf("listen on doctor socket: %w", err)
	}
	chmodErr := os.Chmod(s.cfg.SocketPath, socketMode)
	if chmodErr != nil {
		_ = listener.Close()
		return fmt.Errorf("chmod doctor socket: %w", chmodErr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(doctorRunPath, s.handleRun)
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: doctorReadHeaderTimeout,
	}

	go func() {
		_ = s.httpServer.Serve(listener)
	}()

	return nil
}

func (s *Server) Close(ctx context.Context) error {
	s.closeOnce.Do(func() {
		if s.httpServer != nil {
			if shutdownErr := s.httpServer.Shutdown(ctx); shutdownErr != nil {
				_ = s.httpServer.Close()
				if !errors.Is(shutdownErr, http.ErrServerClosed) {
					s.closeErr = shutdownErr
				}
			}
		}
		if removeErr := os.Remove(s.cfg.SocketPath); removeErr != nil {
			if !errors.Is(removeErr, os.ErrNotExist) {
				s.closeErr = fmt.Errorf("remove doctor socket: %w", removeErr)
			}
		}
	})
	return s.closeErr
}

func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("check stale doctor socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove non-socket doctor path %s", path)
	}
	removeErr := os.Remove(path)
	if removeErr != nil {
		return fmt.Errorf("remove stale doctor socket: %w", removeErr)
	}
	return nil
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
		return
	}
	if !s.beginRun() {
		http.Error(w, "doctor scan is already running\n", http.StatusConflict)
		return
	}
	defer s.endRun()

	header := w.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")

	out := flushWriter{w: w}
	_, err := s.cfg.Runner.RunDoctor(r.Context(), out)
	if err != nil {
		_, _ = fmt.Fprintf(out, "\nDoctor failed: %v\n", err)
	}
}

func (s *Server) beginRun() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *Server) endRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
}

type flushWriter struct {
	w http.ResponseWriter
}

func (w flushWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if flusher, ok := w.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return n, err
}
