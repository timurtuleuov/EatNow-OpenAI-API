package logger

import (
	"log/slog"
	"os"
)

const ServiceName = "eatnow-api"

const (
	KeyComponent  = "component"
	KeyOp         = "op"
	KeyUser       = "user"
	KeyDurationMs = "duration_ms"
	KeyError      = "error"
	KeyVersion    = "version"
	KeyStatus     = "status"
	KeyEnv        = "env"
)

func Init(level slog.Level, version string, isProd bool) *slog.Logger {
	env := "development"
	if isProd {
		env = "production"
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler).With(
		"service", ServiceName,
		KeyVersion, version,
		KeyEnv, env,
	)

	slog.SetDefault(logger)
	return logger
}
