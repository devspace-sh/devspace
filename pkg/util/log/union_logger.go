package log

import (
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

func NewUnionLogger(loggers ...Logger) Logger {
	return &unionLogger{
		Loggers: loggers,
	}
}

type unionLogger struct {
	Loggers []Logger

	level logrus.Level
	m     sync.Mutex
}

func (s *unionLogger) Debug(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.DebugLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Debug(args...)
	}
}

func (s *unionLogger) Debugf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.DebugLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Debugf(format, args...)
	}
}

func (s *unionLogger) Info(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Info(args...)
	}
}

func (s *unionLogger) Infof(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Infof(format, args...)
	}
}

func (s *unionLogger) Warn(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.WarnLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Warn(args...)
	}
}

func (s *unionLogger) Warnf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.WarnLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Warnf(format, args...)
	}
}

func (s *unionLogger) Error(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.ErrorLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Error(args...)
	}
}

func (s *unionLogger) Errorf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.ErrorLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Errorf(format, args...)
	}
}

func (s *unionLogger) Fatal(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.FatalLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Print(logrus.FatalLevel, args...)
	}
	os.Exit(1)
}

func (s *unionLogger) Fatalf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.FatalLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Printf(logrus.FatalLevel, format, args...)
	}
	os.Exit(1)
}

func (s *unionLogger) Done(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Done(args...)
	}
}

func (s *unionLogger) Donef(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return
	}

	for _, l := range s.Loggers {
		l.Donef(format, args...)
	}
}

func (s *unionLogger) Print(level logrus.Level, args ...interface{}) {
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

func (s *unionLogger) Printf(level logrus.Level, format string, args ...interface{}) {
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

func (s *unionLogger) StartWait(message string) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return
	}

	for _, l := range s.Loggers {
		l.StartWait(message)
	}
}

func (s *unionLogger) StopWait() {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return
	}

	for _, l := range s.Loggers {
		l.StopWait()
	}
}

func (s *unionLogger) Writer(level logrus.Level) io.Writer {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < level {
		return ioutil.Discard
	}

	return s
}

func (s *unionLogger) Write(message []byte) (int, error) {
	errs := []error{}
	for _, l := range s.Loggers {
		_, err := l.Writer(logrus.PanicLevel).Write(message)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return len(message), utilerrors.NewAggregate(errs)
}

func (s *unionLogger) WriteString(level logrus.Level, message string) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < level {
		return
	}

	for _, l := range s.Loggers {
		l.WriteString(level, message)
	}
}

func (s *unionLogger) Question(params *survey.QuestionOptions) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < logrus.InfoLevel {
		return "", errors.Errorf("cannot ask question '%s' because log level is too low", params.Question)
	}

	errs := []error{}
	for _, l := range s.Loggers {
		answer, err := l.Question(params)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		return answer, nil
	}

	return "", utilerrors.NewAggregate(errs)
}

func (s *unionLogger) SetLevel(level logrus.Level) {
	s.m.Lock()
	defer s.m.Unlock()

	s.level = level
}

func (s *unionLogger) GetLevel() logrus.Level {
	s.m.Lock()
	defer s.m.Unlock()

	return s.level
}

func (s *unionLogger) WithLevel(level logrus.Level) Logger {
	s.m.Lock()
	defer s.m.Unlock()

	loggers := []Logger{}
	loggers = append(loggers, s.Loggers...)
	return &unionLogger{
		Loggers: loggers,
		level:   level,
	}
}
