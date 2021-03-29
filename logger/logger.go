package logger

import "github.com/sirupsen/logrus"

type Logger struct {
	log *logrus.Logger
}

func NewLogger(service string) *Logger {
	log := logrus.New()
	log.WithField("service", service)

	return &Logger{
		log: log,
	}
}

func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

func (l *Logger) Warning(args ...interface{}) {
	l.log.Warn(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.log.Error(args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.log.Fatal(args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}
