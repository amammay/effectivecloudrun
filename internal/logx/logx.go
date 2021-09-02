package logx

import (
	"context"
	"fmt"
	"github.com/blendle/zapdriver"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type AppLogger struct {
	*zap.Logger
	projectID string
}

func newDevLogger(projectID string) (*AppLogger, error) {
	config := zapdriver.NewDevelopmentConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapLogger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("config.Build(): %v", err)
	}
	return &AppLogger{Logger: zapLogger, projectID: projectID}, nil
}

func newProdLogger(projectID string) (*AppLogger, error) {
	config := zapdriver.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	zapLogger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("config.Build(): %v", err)
	}
	return &AppLogger{
		Logger:    zapLogger,
		projectID: projectID,
	}, nil
}

func NewLogger(projectID string, onCloud bool) (*AppLogger, error) {
	if onCloud {
		return newProdLogger(projectID)
	}
	return newDevLogger(projectID)
}

func (i *AppLogger) WrapTraceContext(ctx context.Context) *zap.SugaredLogger {
	sc := trace.SpanContextFromContext(ctx)
	fields := zapdriver.TraceContext(sc.TraceID().String(), sc.SpanID().String(), sc.IsSampled(), i.projectID)
	setFields := i.With(fields...)
	return setFields.Sugar()
}
