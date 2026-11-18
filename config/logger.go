package config

import (
	"log/slog"
	"os"
)

// NewLogger membuat logger terstruktur (slog). Format JSON dipakai pada
// production agar mudah diproses oleh log aggregator, text untuk development.
func NewLogger(cfg *Config) *slog.Logger {
	level := slog.LevelDebug
	if cfg.IsProduction() {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With("app", cfg.AppName, "env", cfg.AppEnv)
}
