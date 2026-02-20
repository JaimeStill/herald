package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaimeStill/herald/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed:", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info(
		"herald starting",
		"version", cfg.Version,
		"addr", cfg.Server.Addr(),
		"env", cfg.Env(),
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("herald stopped")
}
