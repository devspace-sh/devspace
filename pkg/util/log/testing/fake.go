package testing

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/util/log"
	"io"

	"github.com/loft-sh/devspace/pkg/util/survey"
	fakesurvey "github.com/loft-sh/devspace/pkg/util/survey/testing"
	"github.com/sirupsen/logrus"
)

// FakeLogger just discards every log statement
type FakeLogger struct {
	Survey *fakesurvey.FakeSurvey
	level  logrus.Level
}

// NewFakeLogger returns a new fake logger
func NewFakeLogger() *FakeLogger {
	return &FakeLogger{
		Survey: fakesurvey.NewFakeSurvey(),
	}
}

// Debug implements logger interface
func (d *FakeLogger) Debug(args ...interface{}) {}

// Debugf implements logger interface
func (d *FakeLogger) Debugf(format string, args ...interface{}) {}

// Info implements logger interface
func (d *FakeLogger) Info(args ...interface{}) {}

// Infof implements logger interface
func (d *FakeLogger) Infof(format string, args ...interface{}) {}

// Warn implements logger interface
func (d *FakeLogger) Warn(args ...interface{}) {}

// Warnf implements logger interface
func (d *FakeLogger) Warnf(format string, args ...interface{}) {}

// Error implements logger interface
func (d *FakeLogger) Error(args ...interface{}) {}

// Errorf implements logger interface
func (d *FakeLogger) Errorf(format string, args ...interface{}) {}

// Fatal implements logger interface
func (d *FakeLogger) Fatal(args ...interface{}) {
	d.Panic(args...)
}

// Fatalf implements logger interface
func (d *FakeLogger) Fatalf(format string, args ...interface{}) {
	d.Panicf(format, args...)
}

// Panic implements logger interface
func (d *FakeLogger) Panic(args ...interface{}) {
	panic(fmt.Sprint(args...))
}

// Panicf implements logger interface
func (d *FakeLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

// Done implements logger interface
func (d *FakeLogger) Done(args ...interface{}) {}

// Donef implements logger interface
func (d *FakeLogger) Donef(format string, args ...interface{}) {}

// Fail implements logger interface
func (d *FakeLogger) Fail(args ...interface{}) {}

// Failf implements logger interface
func (d *FakeLogger) Failf(format string, args ...interface{}) {}

// Print implements logger interface
func (d *FakeLogger) Print(level logrus.Level, args ...interface{}) {}

// Printf implements logger interface
func (d *FakeLogger) Printf(level logrus.Level, format string, args ...interface{}) {}

// StartWait implements logger interface
func (d *FakeLogger) StartWait(message string) {}

// StopWait implements logger interface
func (d *FakeLogger) StopWait() {}

// SetLevel implements logger interface
func (d *FakeLogger) SetLevel(level logrus.Level) {
	d.level = level
}

// GetLevel implements logger interface
func (d *FakeLogger) GetLevel() logrus.Level {
	return d.level
}

// Write implements logger interface
func (d *FakeLogger) Write(message []byte) (int, error) {
	return len(message), nil
}

// WriteString implements logger interface
func (d *FakeLogger) WriteString(level logrus.Level, message string) {}

// Question asks a new question
func (d *FakeLogger) Question(params *survey.QuestionOptions) (string, error) {
	return d.Survey.Question(params)
}

func (d *FakeLogger) SetAnswer(answer string) {
	d.Survey.SetNextAnswer(answer)
}

func (d *FakeLogger) Writer(level logrus.Level, raw bool) io.WriteCloser {
	return log.WithNopCloser(io.Discard)
}

func (d *FakeLogger) WithSink(log log.Logger) log.Logger {
	return d
}

func (d *FakeLogger) WithLevel(level logrus.Level) log.Logger {
	return d
}

func (d *FakeLogger) AddSink(log log.Logger) {}

func (d *FakeLogger) WithPrefix(prefix string) log.Logger {
	return d
}

func (d *FakeLogger) WithPrefixColor(prefix, color string) log.Logger {
	return d
}

func (d *FakeLogger) ErrorStreamOnly() log.Logger {
	return d
}
