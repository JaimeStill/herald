package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/lifecycle"
)

type httpServer struct {
	http            *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func newHTTPServer(cfg *config.ServerConfig, handler http.Handler, logger *slog.Logger) *httpServer {
	return &httpServer{
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeoutDuration(),
			WriteTimeout: cfg.WriteTimeoutDuration(),
		},
		logger:          logger.With("system", "http"),
		shutdownTimeout: cfg.ShutdownTimeoutDuration(),
	}
}

func (s *httpServer) Start(lc *lifecycle.Coordinator) error {
	go func() {
		s.logger.Info("server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("server error", "error", err)
		}
	}()

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		s.logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.http.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server shutdown error", "error", err)
		} else {
			s.logger.Info("server shutdown complete")
		}
	})

	return nil
}
