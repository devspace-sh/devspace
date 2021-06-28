package log

import (
	"fmt"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"sync"
	"time"
)

var Colors = []string{
	"blue",
	"green",
	"yellow",
	"magenta",
	"cyan",
	"red",
	"white+b",
}

func NewDefaultPrefixLogger(prefix string, base Logger) Logger {
	return &prefixLogger{
		Logger: base,

		color:  Colors[rand.Intn(len(Colors))],
		prefix: prefix,
	}
}

func NewPrefixLogger(prefix string, color string, base Logger) Logger {
	return &prefixLogger{
		Logger: base,

		color:  color,
		prefix: prefix,
	}
}

type prefixLogger struct {
	Logger

	prefix string
	color  string

	logMutex sync.Mutex
}

func (s *prefixLogger) writeMessage(message string) {
	if os.Getenv(DEVSPACE_LOG_TIMESTAMPS) == "true" {
		now := time.Now()
		if s.color != "" {
			s.WriteString(ansi.Color(formatInt(now.Hour())+":"+formatInt(now.Minute())+":"+formatInt(now.Second())+" ", "white+b") + ansi.Color(s.prefix, s.color) + message)
		} else {
			s.WriteString(formatInt(now.Hour()) + ":" + formatInt(now.Minute()) + ":" + formatInt(now.Second()) + " " + s.prefix + message)
		}
	} else {
		if s.color != "" {
			s.WriteString(ansi.Color(s.prefix, s.color) + message)
		} else {
			s.WriteString(s.prefix + message)
		}
	}
}

func (s *prefixLogger) Debug(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintln(args...))
}

func (s *prefixLogger) Debugf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintf(format, args...) + "\n")
}

func (s *prefixLogger) Info(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintln(args...))
}

func (s *prefixLogger) Infof(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintf(format, args...) + "\n")
}

func (s *prefixLogger) Warn(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage("Warning: " + fmt.Sprintln(args...))
}

func (s *prefixLogger) Warnf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage("Warning: " + fmt.Sprintf(format, args...) + "\n")
}

func (s *prefixLogger) Error(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage("Error: " + fmt.Sprintln(args...))
}

func (s *prefixLogger) Errorf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage("Error: " + fmt.Sprintf(format, args...) + "\n")
}

func (s *prefixLogger) Fatal(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	msg := fmt.Sprintln(args...)
	s.writeMessage("Fatal: " + msg)
	os.Exit(1)
}

func (s *prefixLogger) Fatalf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	msg := fmt.Sprintf(format, args...)
	s.writeMessage("Fatal: " + msg + "\n")
	os.Exit(1)
}

func (s *prefixLogger) Panic(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage("Panic: " + fmt.Sprintln(args...))
	panic(fmt.Sprintln(args...))
}

func (s *prefixLogger) Panicf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage("Panic: " + fmt.Sprintf(format, args...) + "\n")
	panic(fmt.Sprintf(format, args...))
}

func (s *prefixLogger) Done(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintln(args...))
}

func (s *prefixLogger) Donef(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintf(format, args...) + "\n")
}

func (s *prefixLogger) Fail(args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintln(args...))
}

func (s *prefixLogger) Failf(format string, args ...interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.writeMessage(fmt.Sprintf(format, args...) + "\n")
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
	case logrus.PanicLevel:
		s.Panic(args...)
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
	case logrus.PanicLevel:
		s.Panicf(format, args...)
	case logrus.FatalLevel:
		s.Fatalf(format, args...)
	}
}
