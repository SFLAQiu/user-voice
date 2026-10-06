package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger

// Init initializes the global logger.
func Init(level string, file string) error {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("parse log level: %w", err)
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    encCfg,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	if file != "" {
		cfg.OutputPaths = append(cfg.OutputPaths, file)
		cfg.ErrorOutputPaths = append(cfg.ErrorOutputPaths, file)
	}

	logger, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(0))
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	L = logger
	return nil
}

func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}
