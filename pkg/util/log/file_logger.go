package log

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/acarl005/stripansi"

	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// Logdir specifies the relative path to the devspace logs
var Logdir = "./.devspace/logs/"

var logs = map[string]Logger{}
var logsMutex sync.Mutex

var overrideOnce sync.Once

type fileLogger struct {
	logger *logrus.Logger

	m        *sync.Mutex
	level    logrus.Level
	sinks    []Logger
	prefixes []string
}

func GetDevPodFileLogger(devPodName string) Logger {
	return GetFileLogger("dev." + strings.TrimSpace(devPodName))
}

// GetFileLogger returns a logger instance for the specified filename
func GetFileLogger(filename string) Logger {
	filename = strings.TrimSpace(filename)

	logsMutex.Lock()
	defer logsMutex.Unlock()

	log := logs[filename]
	if log == nil {
		newLogger := &fileLogger{
			logger: logrus.New(),
			m:      &sync.Mutex{},
		}
		newLogger.logger.Formatter = &logrus.JSONFormatter{}
		newLogger.logger.SetOutput(&lumberjack.Logger{
			Filename:   Logdir + filename + ".log",
			MaxAge:     12,
			MaxBackups: 4,
			MaxSize:    10 * 1024 * 1024,
		})

		newLogger.SetLevel(GetInstance().GetLevel())
		logs[filename] = newLogger
	}

	return logs[filename]
}

// OverrideRuntimeErrorHandler overrides the standard runtime error handler that logs to stdout
// with a file logger that logs all runtime.HandleErrors to errors.log
func OverrideRuntimeErrorHandler(discard bool) {
	overrideOnce.Do(func() {
		if discard {
			if len(runtime.ErrorHandlers) > 0 {
				runtime.ErrorHandlers[0] = func(err error) {}
			} else {
				runtime.ErrorHandlers = []func(err error){
					func(err error) {},
				}
			}
		} else {
			errorLog := GetFileLogger("errors")
			if len(runtime.ErrorHandlers) > 0 {
				runtime.ErrorHandlers[0] = func(err error) {
					errorLog.Errorf("Runtime error occurred: %s", err)
				}
			} else {
				runtime.ErrorHandlers = []func(err error){
					func(err error) {
						errorLog.Errorf("Runtime error occurred: %s", err)
					},
				}
			}
		}
	})
}

func (f *fileLogger) addPrefixes(message string) string {
	prefix := ""
	for _, p := range f.prefixes {
		prefix += p
	}

	return prefix + message
}

func (f *fileLogger) Debug(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.DebugLevel {
		return
	}

	f.logger.Debug(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Debugf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.DebugLevel {
		return
	}

	f.logger.Debugf(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Info(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Infof(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Warn(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.WarnLevel {
		return
	}

	f.logger.Warn(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Warnf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.WarnLevel {
		return
	}

	f.logger.Warn(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Error(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.ErrorLevel {
		return
	}

	f.logger.Error(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Errorf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.ErrorLevel {
		return
	}

	f.logger.Error(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Fatal(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.FatalLevel {
		return
	}

	f.logger.Fatal(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Fatalf(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.FatalLevel {
		return
	}

	f.logger.Fatal(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Done(args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprint(args...))))
}

func (f *fileLogger) Donef(format string, args ...interface{}) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	f.logger.Info(f.addPrefixes(stripEscapeSequences(fmt.Sprintf(format, args...))))
}

func (f *fileLogger) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		f.Info(args...)
	case logrus.DebugLevel:
		f.Debug(args...)
	case logrus.WarnLevel:
		f.Warn(args...)
	case logrus.ErrorLevel:
		f.Error(args...)
	case logrus.FatalLevel:
		f.Fatal(args...)
	}
}

func (f *fileLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		f.Infof(format, args...)
	case logrus.DebugLevel:
		f.Debugf(format, args...)
	case logrus.WarnLevel:
		f.Warnf(format, args...)
	case logrus.ErrorLevel:
		f.Errorf(format, args...)
	case logrus.FatalLevel:
		f.Fatalf(format, args...)
	}
}

func (f *fileLogger) StartWait(message string) {
	// Noop operation
}

func (f *fileLogger) StopWait() {
	// Noop operation
}

func (f *fileLogger) SetLevel(level logrus.Level) {
	f.m.Lock()
	defer f.m.Unlock()

	f.level = level
}

func (f *fileLogger) GetLevel() logrus.Level {
	f.m.Lock()
	defer f.m.Unlock()

	return f.level
}

func (f *fileLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < level {
		return &NopCloser{io.Discard}
	}

	return &NopCloser{f}
}

func (f *fileLogger) Write(message []byte) (int, error) {
	return f.logger.Out.Write(message)
}

func (f *fileLogger) WriteString(level logrus.Level, message string) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.level < logrus.InfoLevel {
		return
	}

	_, _ = f.logger.Out.Write([]byte(stripEscapeSequences(message)))
}

func stripEscapeSequences(str string) string {
	return stripansi.Strip(strings.TrimSpace(str))
}

func (f *fileLogger) Question(params *survey.QuestionOptions) (string, error) {
	return "", errors.New("questions in file logger not supported")
}

// WithLevel implements logger interface
func (f *fileLogger) WithLevel(level logrus.Level) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := *f
	n.m = &sync.Mutex{}
	n.level = level
	return &n
}

func (f *fileLogger) WithSink(log Logger) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := *f
	n.m = &sync.Mutex{}
	n.sinks = append(n.sinks, log)
	return &n
}

func (f *fileLogger) AddSink(log Logger) {
	f.m.Lock()
	defer f.m.Unlock()

	f.sinks = append(f.sinks, log)
}

func (f *fileLogger) WithPrefix(prefix string) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := *f
	n.m = &sync.Mutex{}
	n.prefixes = append(n.prefixes, prefix)
	return &n
}

func (f *fileLogger) WithPrefixColor(prefix, color string) Logger {
	f.m.Lock()
	defer f.m.Unlock()

	n := *f
	n.m = &sync.Mutex{}
	n.prefixes = append(n.prefixes, prefix)
	return &n
}

func (f *fileLogger) ErrorStreamOnly() Logger {
	return f
}
