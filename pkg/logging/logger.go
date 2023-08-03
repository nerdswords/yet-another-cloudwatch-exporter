package logging

import (
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Logger interface {
	Info(message string, keyvals ...interface{})
	Debug(message string, keyvals ...interface{})
	Error(err error, message string, keyvals ...interface{})
	Warn(message string, keyvals ...interface{})
	With(keyvals ...interface{}) Logger
	IsDebugEnabled() bool
}

type gokitLogger struct {
	logger       log.Logger
	debugEnabled bool
}

func NewLogger(format string, debugEnabled bool, keyvals ...interface{}) Logger {
	var logger log.Logger
	if format == "json" {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	} else {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	}

	if debugEnabled {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.Caller(4))
	logger = log.With(logger, keyvals...)

	return gokitLogger{
		logger:       logger,
		debugEnabled: debugEnabled,
	}
}

func NewNopLogger() Logger {
	return gokitLogger{logger: log.NewNopLogger()}
}

func (g gokitLogger) Debug(message string, keyvals ...interface{}) {
	if g.debugEnabled {
		kv := []interface{}{"msg", message}
		kv = append(kv, keyvals...)
		level.Debug(g.logger).Log(kv...)
	}
}

func (g gokitLogger) Info(message string, keyvals ...interface{}) {
	kv := []interface{}{"msg", message}
	kv = append(kv, keyvals...)
	level.Info(g.logger).Log(kv...)
}

func (g gokitLogger) Error(err error, message string, keyvals ...interface{}) {
	kv := []interface{}{"msg", message, "err", err}
	kv = append(kv, keyvals...)
	level.Error(g.logger).Log(kv...)
}

func (g gokitLogger) Warn(message string, keyvals ...interface{}) {
	kv := []interface{}{"msg", message}
	kv = append(kv, keyvals...)
	level.Warn(g.logger).Log(kv...)
}

func (g gokitLogger) With(keyvals ...interface{}) Logger {
	return gokitLogger{
		logger:       log.With(g.logger, keyvals...),
		debugEnabled: g.debugEnabled,
	}
}

func (g gokitLogger) IsDebugEnabled() bool {
	return g.debugEnabled
}
