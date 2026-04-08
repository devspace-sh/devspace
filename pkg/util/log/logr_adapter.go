package log

import (
	"github.com/go-logr/logr"
)

// LogrSink is an adapter that wraps our Logger interface to implement logr.LogSink
type LogrSink struct {
	logger Logger
	name   string
}

// Init receives optional information about the logr library
func (l *LogrSink) Init(info logr.RuntimeInfo) {}

// Enabled tests whether this LogSink is enabled at the specified V-level
func (l *LogrSink) Enabled(level int) bool {
	return true
}

// Info logs a non-error message with the given key/value pairs as context
func (l *LogrSink) Info(level int, msg string, keysAndValues ...any) {
	if level > 0 {
		l.logger.Debugf("%s %v", msg, keysAndValues)
	} else {
		l.logger.Infof("%s %v", msg, keysAndValues)
	}
}

// Error logs an error, with the given message and key/value pairs as context
func (l *LogrSink) Error(err error, msg string, keysAndValues ...any) {
	if err != nil {
		l.logger.Errorf("%s: %v %v", msg, err, keysAndValues)
	} else {
		l.logger.Errorf("%s %v", msg, keysAndValues)
	}
}

// WithValues returns a new LogSink with additional key/value pairs
func (l *LogrSink) WithValues(keysAndValues ...any) logr.LogSink {
	// Our logger doesn't support key-value pairs, just return the same sink
	return l
}

// WithName returns a new LogSink with the specified name appended
func (l *LogrSink) WithName(name string) logr.LogSink {
	newName := name
	if l.name != "" {
		newName = l.name + "/" + name
	}
	return &LogrSink{
		logger: l.logger.WithPrefix("[" + name + "] "),
		name:   newName,
	}
}

// ToLogr converts a Logger to a logr.Logger
func ToLogr(logger Logger) logr.Logger {
	return logr.New(&LogrSink{logger: logger})
}
