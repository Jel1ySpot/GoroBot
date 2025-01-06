package logger

import "log"

type Inst interface {
	SetLogLevel(LogLevel)
	Info(format string, args ...any)
	Warning(format string, args ...any)
	Error(format string, args ...any)
	Debug(format string, args ...any)
	Dump(dumped []byte, format string, args ...any)
}

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warning
	Error
)

type DefaultLogger struct {
	LogLevel LogLevel
}

func (l *DefaultLogger) SetLogLevel(level LogLevel) {
	l.LogLevel = level
}

func (l *DefaultLogger) Info(format string, args ...any) {
	if l.LogLevel <= Info {
		log.Printf(format, args...)
	}
}

func (l *DefaultLogger) Warning(format string, args ...any) {
	if l.LogLevel <= Warning {
		log.Printf(format, args...)
	}
}

func (l *DefaultLogger) Error(format string, args ...any) {
	if l.LogLevel <= Error {
		log.Printf(format, args...)
	}
}

func (l *DefaultLogger) Debug(format string, args ...any) {
	if l.LogLevel <= Debug {
		log.Printf(format, args...)
	}
}

func (l *DefaultLogger) Dump(dumped []byte, format string, args ...any) {
	log.Printf(format, args...)
}
