package log

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

var Colors = []string{
	"blue",
	"green",
	"yellow",
	"magenta",
	"cyan",
	"white+b",
}

func NewDefaultPrefixLogger(prefix string, base Logger) Logger {
	hashNumber := int(hash.StringToNumber(prefix))
	if hashNumber < 0 {
		hashNumber = hashNumber * -1
	}

	return &prefixLogger{
		base:   base,
		level:  base.GetLevel(),
		color:  Colors[hashNumber%len(Colors)],
		prefix: prefix,
	}
}

func NewPrefixLogger(prefix string, color string, base Logger) Logger {
	return &prefixLogger{
		base: base,

		level: base.GetLevel(),

		color:  color,
		prefix: prefix,
	}
}

type prefixLogger struct {
	base Logger

	level logrus.Level

	prefix string
	color  string

	m sync.Mutex
}

func (s *prefixLogger) Children() []Logger {
	return []Logger{s.base}
}

func (s *prefixLogger) WithLevel(level logrus.Level) Logger {
	s.m.Lock()
	defer s.m.Unlock()

	return &prefixLogger{
		base:   s.base,
		level:  level,
		prefix: s.prefix,
		color:  s.color,
	}
}

func (s *prefixLogger) SetLevel(level logrus.Level) {
	s.m.Lock()
	defer s.m.Unlock()

	s.level = level
}

func (s *prefixLogger) GetLevel() logrus.Level {
	s.m.Lock()
	defer s.m.Unlock()

	return s.level
}

func (s *prefixLogger) writeMessage(level logrus.Level, message string) {
	if s.level >= level {
		prefix := ""
		if s.color != "" {
			prefix = ansi.Color(s.prefix, s.color)
		} else {
			prefix = s.prefix
		}

		s.base.Print(level, prefix+strings.TrimSpace(message))
	}
}

func (s *prefixLogger) Debug(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.DebugLevel, fmt.Sprintln(args...))
}

func (s *prefixLogger) Debugf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.DebugLevel, fmt.Sprintf(format, args...)+"\n")
}

func (s *prefixLogger) Info(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.InfoLevel, fmt.Sprintln(args...))
}

func (s *prefixLogger) Infof(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.InfoLevel, fmt.Sprintf(format, args...)+"\n")
}

func (s *prefixLogger) Warn(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.WarnLevel, "Warning: "+fmt.Sprintln(args...))
}

func (s *prefixLogger) Warnf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.WarnLevel, "Warning: "+fmt.Sprintf(format, args...)+"\n")
}

func (s *prefixLogger) Error(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.ErrorLevel, "Error: "+fmt.Sprintln(args...))
}

func (s *prefixLogger) Errorf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.ErrorLevel, "Error: "+fmt.Sprintf(format, args...)+"\n")
}

func (s *prefixLogger) Fatal(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	msg := fmt.Sprintln(args...)
	s.writeMessage(logrus.FatalLevel, "Fatal: "+msg)
	os.Exit(1)
}

func (s *prefixLogger) Fatalf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	msg := fmt.Sprintf(format, args...)
	s.writeMessage(logrus.FatalLevel, "Fatal: "+msg+"\n")
	os.Exit(1)
}

func (s *prefixLogger) Done(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.InfoLevel, fmt.Sprintln(args...))
}

func (s *prefixLogger) Donef(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(logrus.InfoLevel, fmt.Sprintf(format, args...)+"\n")
}

func (s *prefixLogger) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Info(args...)
	case logrus.DebugLevel:
		s.Debug(args...)
	case logrus.WarnLevel:
		s.Warn(args...)
	case logrus.ErrorLevel:
		s.Error(args...)
	case logrus.FatalLevel:
		s.Fatal(args...)
	}
}

func (s *prefixLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		s.Infof(format, args...)
	case logrus.DebugLevel:
		s.Debugf(format, args...)
	case logrus.WarnLevel:
		s.Warnf(format, args...)
	case logrus.ErrorLevel:
		s.Errorf(format, args...)
	case logrus.FatalLevel:
		s.Fatalf(format, args...)
	}
}

func (s *prefixLogger) Writer(level logrus.Level) io.WriteCloser {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < level {
		return WithNopCloser(ioutil.Discard)
	}

	reader, writer := io.Pipe()
	go func() {
		sa := scanner.NewScanner(reader)
		for sa.Scan() {
			s.Print(level, sa.Text())
		}
	}()

	return writer
}

func WithNopCloser(writer io.Writer) io.WriteCloser {
	return &NopCloser{writer}
}

type NopCloser struct {
	io.Writer
}

func (NopCloser) Close() error { return nil }

func (s *prefixLogger) Write(message []byte) (int, error) {
	s.m.Lock()
	defer s.m.Unlock()

	s.base.WriteString(logrus.FatalLevel, string(message))
	return len(message), nil
}

func (s *prefixLogger) WriteString(level logrus.Level, message string) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < level {
		return
	}

	s.base.WriteString(level, message)
}

func (s *prefixLogger) Question(params *survey.QuestionOptions) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return "", errors.Errorf("cannot ask question '%s' because log level is too low", params.Question)
	}

	return s.base.Question(params)
}
