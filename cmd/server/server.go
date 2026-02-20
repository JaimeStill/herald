package main

import (
	"time"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
)

type Server struct {
	infra   *infrastructure.Infrastructure
	modules *Modules
	http    *httpServer
}

func NewServer(cfg *config.Config) (*Server, error) {
	infra, err := infrastructure.New(cfg)
	if err != nil {
		return nil, err
	}

	modules, err := NewModules(infra, cfg)
	if err != nil {
		return nil, err
	}

	router := buildRouter(infra)
	modules.Mount(router)

	infra.Logger.Info(
		"server initialized",
		"addr", cfg.Server.Addr(),
		"version", cfg.Version,
	)

	return &Server{
		infra:   infra,
		modules: modules,
		http:    newHTTPServer(&cfg.Server, router, infra.Logger),
	}, nil
}

func (s *Server) Start() error {
	s.infra.Logger.Info("starting service")

	if err := s.infra.Start(); err != nil {
		return err
	}

	if err := s.http.Start(s.infra.Lifecycle); err != nil {
		return err
	}

	go func() {
		s.infra.Lifecycle.WaitForStartup()
		s.infra.Logger.Info("all subsystems ready")
	}()

	return nil
}

func (s *Server) Shutdown(timeout time.Duration) error {
	s.infra.Logger.Info("initiating shutdown")
	return s.infra.Lifecycle.Shutdown(timeout)
}
