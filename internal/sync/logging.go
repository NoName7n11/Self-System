package sync

import "log/slog"

func syncLogger() *slog.Logger {
	return slog.Default().With("component", "sync")
}
