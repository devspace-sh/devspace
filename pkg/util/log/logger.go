package log

import (
	"io"

	"github.com/Sirupsen/logrus"
)

// Logger defines the common logging interface
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})

	Done(args ...interface{})
	Donef(format string, args ...interface{})

	Fail(args ...interface{})
	Failf(format string, args ...interface{})

	With(object interface{}) *LoggerEntry

	Print(level logrus.Level, args ...interface{})
	Printf(level logrus.Level, format string, args ...interface{})

	SetLevel(level logrus.Level)
	GetStream() io.Writer

	printWithContext(fnType logFunctionType, context []interface{}, args ...interface{})
	printWithContextf(fnType logFunctionType, context []interface{}, format string, args ...interface{})
}
