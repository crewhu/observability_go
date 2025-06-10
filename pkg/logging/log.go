package logging

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	otellog "go.opentelemetry.io/otel/log"
)

type (
	LogLevel slog.Level
)

var (
	logger *slog.Logger

	LogLevelInfo  LogLevel = LogLevel(slog.LevelInfo)
	LogLevelWarn           = LogLevel(slog.LevelWarn)
	LogLevelError          = LogLevel(slog.LevelError)
	LogLevelDebug          = LogLevel(slog.LevelDebug)
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func (l LogLevel) OtelString() otellog.Severity {
	switch l {
	case LogLevelInfo:
		return otellog.SeverityInfo
	case LogLevelWarn:
		return otellog.SeverityWarn
	case LogLevelError:
		return otellog.SeverityError
	case LogLevelDebug:
		return otellog.SeverityDebug
	default:
		return otellog.SeverityUndefined
	}
}

func init() {
	logger = slog.Default()
}

func SetLoggingLevel(level LogLevel) {
	ConfigureLogger(level)
	slog.Info("Logging level set", slog.String("level", level.String()))
}

func Log(ctx context.Context, level LogLevel, msg string, opts ...any) {
	args := []any{}
	tags := Tags{}
	maps.Copy(tags, getTags(ctx))
	for _, opt := range opts {
		switch opt := opt.(type) {
		case Tags:
			maps.Copy(tags, opt)
		default:
			args = append(args, opt)
		}
	}
	printf(ctx, level, tags, msg, args...)
}

func Debug(ctx context.Context, msg string, opts ...any) {
	Log(ctx, LogLevelDebug, msg, opts...)
}

func Info(ctx context.Context, msg string, opts ...any) {
	Log(ctx, LogLevelInfo, msg, opts...)
}

func Warn(ctx context.Context, msg string, opts ...any) {
	Log(ctx, LogLevelWarn, msg, opts...)
}

func Error(ctx context.Context, msg string, opts ...any) {
	Log(ctx, LogLevelError, msg, opts...)
}

func printf(ctx context.Context, level LogLevel, t Tags, msg string, v ...any) {
	ctxLogger := GetLoggerFromContext(ctx)

	formattedMsg := msg
	if len(v) > 0 {
		formattedMsg = fmt.Sprintf(msg, v...)
	}

	ctxLogger.LogAttrs(ctx, slog.Level(level), formattedMsg)
	otelPrintf(ctx, level, formattedMsg)
}

func otelPrintf(ctx context.Context, level LogLevel, msg string) {
	if otelLogger == nil {
		return
	}

	otelRecord := GetOtelLoggerFromContext(ctx)
	otelRecord.SetBody(otellog.StringValue(msg))
	otelRecord.SetSeverity(level.OtelString())
	otelLogger.Emit(ctx, otelRecord)
}

func Err(ctx context.Context, err error, tags ...Tags) {
	t := Tags{}
	maps.Copy(t, getTags(ctx))
	for _, tag := range tags {
		t = t.Merge(tag)
	}
	t = t.Merge(Tags{"error": true})
	printf(ctx, LogLevelError, t, err.Error())
}
