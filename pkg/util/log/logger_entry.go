package log

import (
	"github.com/Sirupsen/logrus"
)

// Level type
type logFunctionType uint32

const (
	panicFn logFunctionType = iota
	fatalFn
	errorFn
	warnFn
	infoFn
	debugFn
	failFn
	doneFn
)

// LoggerEntry defines an entry to the logger
type LoggerEntry struct {
	logger  Logger
	context []interface{}
}

// Debug prints debug information
func (l *LoggerEntry) Debug(args ...interface{}) {
	l.logger.printWithContext(debugFn, l.context, args...)
}

// Debugf prints formatted debug information
func (l *LoggerEntry) Debugf(format string, args ...interface{}) {
	l.logger.printWithContextf(debugFn, l.context, format, args...)
}

// Info prints info information
func (l *LoggerEntry) Info(args ...interface{}) {
	l.logger.printWithContext(infoFn, l.context, args...)
}

// Infof prints formatted info information
func (l *LoggerEntry) Infof(format string, args ...interface{}) {
	l.logger.printWithContextf(infoFn, l.context, format, args...)
}

// Warn prints warn information
func (l *LoggerEntry) Warn(args ...interface{}) {
	l.logger.printWithContext(warnFn, l.context, args...)
}

// Warnf prints formatted warn information
func (l *LoggerEntry) Warnf(format string, args ...interface{}) {
	l.logger.printWithContextf(warnFn, l.context, format, args...)
}

// Error prints error information
func (l *LoggerEntry) Error(args ...interface{}) {
	l.logger.printWithContext(errorFn, l.context, args...)
}

// Errorf prints formatted error information
func (l *LoggerEntry) Errorf(format string, args ...interface{}) {
	l.logger.printWithContextf(errorFn, l.context, format, args...)
}

// Fatal prints fatal error information
func (l *LoggerEntry) Fatal(args ...interface{}) {
	l.logger.printWithContext(fatalFn, l.context, args...)
}

// Fatalf prints formatted fatal error information
func (l *LoggerEntry) Fatalf(format string, args ...interface{}) {
	l.logger.printWithContextf(fatalFn, l.context, format, args...)
}

// Panic prints panic information
func (l *LoggerEntry) Panic(args ...interface{}) {
	l.logger.printWithContext(panicFn, l.context, args...)
}

// Panicf prints formatted panic information
func (l *LoggerEntry) Panicf(format string, args ...interface{}) {
	l.logger.printWithContextf(panicFn, l.context, format, args...)
}

// Done prints info information
func (l *LoggerEntry) Done(args ...interface{}) {
	l.logger.printWithContext(doneFn, l.context, args...)
}

// Donef prints formatted info information
func (l *LoggerEntry) Donef(format string, args ...interface{}) {
	l.logger.printWithContextf(doneFn, l.context, format, args...)
}

// Fail prints error information
func (l *LoggerEntry) Fail(args ...interface{}) {
	l.logger.printWithContext(failFn, l.context, args...)
}

// Failf prints formatted error information
func (l *LoggerEntry) Failf(format string, args ...interface{}) {
	l.logger.printWithContextf(failFn, l.context, format, args...)
}

// Print prints information
func (l *LoggerEntry) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		l.Info(args...)
	case logrus.DebugLevel:
		l.Debug(args...)
	case logrus.WarnLevel:
		l.Warn(args...)
	case logrus.ErrorLevel:
		l.Error(args...)
	case logrus.PanicLevel:
		l.Panic(args...)
	case logrus.FatalLevel:
		l.Fatal(args...)
	}
}

// Printf prints formatted information
func (l *LoggerEntry) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		l.Infof(format, args...)
	case logrus.DebugLevel:
		l.Debugf(format, args...)
	case logrus.WarnLevel:
		l.Warnf(format, args...)
	case logrus.ErrorLevel:
		l.Errorf(format, args...)
	case logrus.PanicLevel:
		l.Panicf(format, args...)
	case logrus.FatalLevel:
		l.Fatalf(format, args...)
	}
}

// With adds context information to the entry
func (l *LoggerEntry) With(obj interface{}) *LoggerEntry {
	l.context = append(l.context, obj)

	return l
}
