package logging

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"novel-agent-runtime/internal/config"
)

type contextKey struct{}

func New(cfg config.LoggingConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if cfg.Level != "" {
		if err := level.Set(cfg.Level); err != nil {
			return nil, fmt.Errorf("parse logging.level: %w", err)
		}
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeDuration = zapcore.MillisDurationEncoder
	encoderCfg.TimeKey = "ts"
	encoderCfg.MessageKey = "msg"
	encoderCfg.LevelKey = "level"
	encoderCfg.CallerKey = "caller"

	zapCfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       cfg.Development,
		Encoding:          cfg.Encoding,
		EncoderConfig:     encoderCfg,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableStacktrace: !cfg.Development,
	}
	if zapCfg.Encoding == "" {
		zapCfg.Encoding = "json"
	}
	if cfg.Development {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	}
	logger, err := zapCfg.Build(zap.AddCaller())
	if err != nil {
		return nil, err
	}
	return logger.Named("novelrt"), nil
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.NewNop()
	}
	if logger, ok := ctx.Value(contextKey{}).(*zap.Logger); ok && logger != nil {
		return logger
	}
	return zap.NewNop()
}

func NewRequestID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
