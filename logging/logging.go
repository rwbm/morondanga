package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	isDev  bool
	level  int
	format string
)

// Get returns a logger instance.
func Get() *zap.Logger {
	if logger != nil {
		return logger
	}

	logger = configLogger(level, isDev, format)
	return logger
}

// GetWithConfig returns a logger instance.
func GetWithConfig(level int, isDev bool, format string) *zap.Logger {
	if logger != nil {
		return logger
	}

	logger = configLogger(level, isDev, format)
	return logger
}

func configLogger(level int, isDev bool, format string) *zap.Logger {
	var encoderCfg zapcore.EncoderConfig
	if isDev {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
	}

	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zapcore.Level(level)),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          format,
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stderr",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
	}
	logger = zap.Must(config.Build())
	return logger
}
