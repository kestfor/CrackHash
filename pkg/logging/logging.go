package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type LoggerConfig struct {
	Level  string `yaml:"level"`
	IsJSON bool   `yaml:"is_json"`
}

func InitLogger(cfg *LoggerConfig, attrs ...slog.Attr) {
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

	options := log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Level:           log.Level(level),
	}
	if cfg.IsJSON {
		options.Formatter = log.JSONFormatter
	} else {
		options.Formatter = log.TextFormatter
	}

	h := log.NewWithOptions(os.Stderr, options)
	slog.SetDefault(slog.New(h.WithAttrs(attrs)))
}
