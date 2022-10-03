package logger

import (
	"go.uber.org/zap"
)

type Logger struct {
	log *zap.SugaredLogger
}

func NewLogger(service string) *Logger {
	log, _ := zap.NewProduction()
	return &Logger{
		log: log.Named(service).Sugar(),
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
