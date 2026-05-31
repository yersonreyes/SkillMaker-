package logger

import (
	"log/slog"
	"os"
)

// New creates a configured *slog.Logger.
// When env is "production" it uses a JSON handler; otherwise a text handler
// suitable for development terminals.
// The level string is parsed with slog.Level.UnmarshalText (e.g. "debug",
// "info", "warn", "error").
func New(level, appEnv string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if appEnv == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
