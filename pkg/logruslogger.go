package exporter

import log "github.com/sirupsen/logrus"

type logrusLogger struct {
	logger *log.Logger
}

func NewLogrusLogger(logger *log.Logger) logrusLogger {
	return logrusLogger{logger}
}

func (l logrusLogger) Info(message string, args ...interface{}) {
	l.logger.Infof(message, args...)
}

func (l logrusLogger) Debug(message string, args ...interface{}) {
	l.logger.Debugf(message, args...)
}

func (l logrusLogger) Error(err error, message string, args ...interface{}) {
	l.logger.WithError(err).Errorf(message, args...)
}

func (l logrusLogger) Warn(message string, args ...interface{}) {
	l.logger.Warnf(message, args...)
}

func (l logrusLogger) IsDebugEnabled() bool {
	return l.logger.IsLevelEnabled(log.DebugLevel)
}
