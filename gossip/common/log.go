package common

import "log/slog"

const (
	// custom log level used for end-to-end testing and benchmarking
	LevelTest = slog.Level(-8)
)
