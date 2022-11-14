package log

import (
	"io"
	"os"

	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/sirupsen/logrus"
)

var _ Logger = &DiscardLogger{}

// DiscardLogger just discards every log statement
type DiscardLogger struct{}

// WithLevel implements logger interface
func (d *DiscardLogger) WithLevel(level logrus.Level) Logger {
	return &DiscardLogger{}
}

// Debug implements logger interface
func (d *DiscardLogger) Debug(args ...interface{}) {}

// Debugf implements logger interface
func (d *DiscardLogger) Debugf(format string, args ...interface{}) {}

// Info implements logger interface
func (d *DiscardLogger) Info(args ...interface{}) {}

// Infof implements logger interface
func (d *DiscardLogger) Infof(format string, args ...interface{}) {}

// Warn implements logger interface
func (d *DiscardLogger) Warn(args ...interface{}) {}

// Warnf implements logger interface
func (d *DiscardLogger) Warnf(format string, args ...interface{}) {}

// Error implements logger interface
func (d *DiscardLogger) Error(args ...interface{}) {}

// Errorf implements logger interface
func (d *DiscardLogger) Errorf(format string, args ...interface{}) {}

// Fatal implements logger interface
func (d *DiscardLogger) Fatal(args ...interface{}) {
	os.Exit(1)
}

func (d *DiscardLogger) Children() []Logger {
	return nil
}

// Fatalf implements logger interface
func (d *DiscardLogger) Fatalf(format string, args ...interface{}) {
	os.Exit(1)
}

// Done implements logger interface
func (d *DiscardLogger) Done(args ...interface{}) {}

// Donef implements logger interface
func (d *DiscardLogger) Donef(format string, args ...interface{}) {}

// Fail implements logger interface
func (d *DiscardLogger) Fail(args ...interface{}) {}

// Failf implements logger interface
func (d *DiscardLogger) Failf(format string, args ...interface{}) {}

// Print implements logger interface
func (d *DiscardLogger) Print(level logrus.Level, args ...interface{}) {}

// Printf implements logger interface
func (d *DiscardLogger) Printf(level logrus.Level, format string, args ...interface{}) {}

// StartWait implements logger interface
func (d *DiscardLogger) StartWait(message string) {}

// StopWait implements logger interface
func (d *DiscardLogger) StopWait() {}

// SetLevel implements logger interface
func (d *DiscardLogger) SetLevel(level logrus.Level) {}

// GetLevel implements logger interface
func (d *DiscardLogger) GetLevel() logrus.Level { return logrus.FatalLevel }

// Write implements logger interface
func (d *DiscardLogger) Write(message []byte) (int, error) {
	return len(message), nil
}

func (d *DiscardLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	return &NopCloser{io.Discard}
}

// WriteString implements logger interface
func (d *DiscardLogger) WriteString(level logrus.Level, message string) {}

// Question asks a new question
func (d *DiscardLogger) Question(params *survey.QuestionOptions) (string, error) {
	return "", SurveyError{}
}

// SurveyError is used to identify errors where questions were asked in the discard logger
type SurveyError struct{}

// Error implements error interface
func (s SurveyError) Error() string {
	return "Asking questions is not possible in silenced mode"
}

func (d *DiscardLogger) WithSink(log Logger) Logger {
	return d
}

func (d *DiscardLogger) AddSink(log Logger) {}

func (d *DiscardLogger) WithPrefix(prefix string) Logger {
	return d
}

func (d *DiscardLogger) WithPrefixColor(prefix, color string) Logger {
	return d
}

func (d *DiscardLogger) ErrorStreamOnly() Logger {
	return d
}
