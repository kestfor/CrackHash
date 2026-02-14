package logging

import (
	"log/slog"
	"os"
	"strings"
)

type LoggerConfig struct {
	Level  string `yaml:"level"`
	IsJSON bool   `yaml:"is_json"`
}

func InitLogger(cfg *LoggerConfig, attrs ...slog.Attr) {
	var h slog.Handler

	var level slog.Level

	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	if cfg.IsJSON {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		h = slog.Default().Handler()
	}

	slog.SetDefault(slog.New(h.WithAttrs(attrs)))
	lvl := new(slog.LevelVar)
	lvl.Set(level)
}
