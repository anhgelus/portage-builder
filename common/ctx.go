package common

import (
	"context"
	"log/slog"
)

type key uint

const (
	loggerKey key = iota
)

func NewLoggerContext(parent context.Context, log *slog.Logger) context.Context {
	return context.WithValue(parent, loggerKey, log)
}

func ContextLogger(ctx context.Context) *slog.Logger {
	log, ok := ctx.Value(loggerKey).(*slog.Logger)
	if ok {
		return log
	}
	return slog.Default()
}
