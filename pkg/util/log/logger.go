package log

import (
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/sirupsen/logrus"
	"io"
)

// Level type
type logFunctionType uint32

const (
	fatalFn logFunctionType = iota
	errorFn
	warnFn
	infoFn
	debugFn
	doneFn
)

// Logger defines the common logging interface
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})

	Done(args ...interface{})
	Donef(format string, args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})

	Print(level logrus.Level, args ...interface{})
	Printf(level logrus.Level, format string, args ...interface{})

	Writer(level logrus.Level, raw bool) io.WriteCloser
	WriteString(level logrus.Level, message string)

	Question(params *survey.QuestionOptions) (string, error)

	SetLevel(level logrus.Level)
	GetLevel() logrus.Level

	// WithLevel creates a new logger with the given level
	WithLevel(level logrus.Level) Logger
	ErrorStreamOnly() Logger
	WithPrefix(prefix string) Logger
	WithPrefixColor(prefix, color string) Logger
	WithSink(sink Logger) Logger
	AddSink(sink Logger)
}
