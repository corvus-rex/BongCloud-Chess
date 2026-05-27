package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// In development: human-readable console output with colour and caller info.
// In production: JSON output, no caller info, sampling enabled.
func New(level, encoding, appName, version string) (*zap.Logger, error) {
	logLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	var cfg zap.Config

	if encoding == "console" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	cfg.Level = zap.NewAtomicLevelAt(logLevel)
	cfg.Encoding = encoding
	cfg.InitialFields = map[string]interface{}{
		"service": appName,
		"version": version,
	}

	log, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(0))
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return log, nil
}

func Must(level, encoding, appName, version string) *zap.Logger {
	log, err := New(level, encoding, appName, version)
	if err != nil {
		panic(err)
	}
	return log
}
