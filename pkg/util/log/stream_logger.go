package log

import (
	"encoding/json"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/env"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/acarl005/stripansi"
	goansi "github.com/k0kubun/go-ansi"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/loft-sh/devspace/pkg/util/terminal"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const DevSpaceLogTimestamps = "DEVSPACE_LOG_TIMESTAMPS"

var stdout = goansi.NewAnsiStdout()

type Format int

const (
	TextFormat Format = iota
	TimeFormat Format = iota
	JsonFormat Format = iota
	RawFormat  Format = iota
)

func NewStdoutLogger(reader io.Reader, writer io.Writer, level logrus.Level) Logger {
	isTerminal, _ := terminal.SetupTTY(reader, writer)
	return &StreamLogger{
		m:          sync.Mutex{},
		level:      level,
		format:     TextFormat,
		isTerminal: isTerminal,
		stream:     writer,
		survey:     survey.NewSurvey(),
	}
}

func NewStreamLogger(writer io.Writer, level logrus.Level) Logger {
	return &StreamLogger{
		m:          sync.Mutex{},
		level:      level,
		format:     TextFormat,
		isTerminal: false,
		stream:     writer,
		survey:     survey.NewSurvey(),
	}
}

func NewStreamLoggerWithFormat(writer io.Writer, level logrus.Level, format Format) Logger {
	return &StreamLogger{
		m:          sync.Mutex{},
		level:      level,
		isTerminal: false,
		format:     format,
		stream:     writer,
		survey:     survey.NewSurvey(),
	}
}

type StreamLogger struct {
	m     sync.Mutex
	level logrus.Level

	format     Format
	isTerminal bool
	stream     io.Writer

	survey survey.Survey
}

type Line struct {
	// Time is when this log message occurred
	Time time.Time `json:"time,omitempty"`

	// Message is when the message of the log message
	Message string `json:"message,omitempty"`

	// Level is the log level this message has used
	Level logrus.Level `json:"level,omitempty"`
}

type fnTypeInformation struct {
	tag      string
	color    string
	logLevel logrus.Level
}

var fnTypeInformationMap = map[logFunctionType]*fnTypeInformation{
	debugFn: {
		tag:      "debug ",
		color:    "green+b",
		logLevel: logrus.DebugLevel,
	},
	infoFn: {
		tag:      "info ",
		color:    "cyan+b",
		logLevel: logrus.InfoLevel,
	},
	warnFn: {
		tag:      "warn ",
		color:    "red+b",
		logLevel: logrus.WarnLevel,
	},
	errorFn: {
		tag:      "error ",
		color:    "red+b",
		logLevel: logrus.ErrorLevel,
	},
	fatalFn: {
		tag:      "fatal ",
		color:    "red+b",
		logLevel: logrus.FatalLevel,
	},
	doneFn: {
		tag:      "done ",
		color:    "green+b",
		logLevel: logrus.InfoLevel,
	},
}

func formatInt(i int) string {
	formatted := strconv.Itoa(i)
	if len(formatted) == 1 {
		formatted = "0" + formatted
	}
	return formatted
}

func (s *StreamLogger) GetFormat() Format {
	s.m.Lock()
	defer s.m.Unlock()

	return s.format
}

func (s *StreamLogger) WithLevel(level logrus.Level) Logger {
	s.m.Lock()
	defer s.m.Unlock()

	return &StreamLogger{
		stream:     s.stream,
		format:     s.format,
		isTerminal: s.isTerminal,
		level:      level,
		survey:     survey.NewSurvey(),
	}
}

func (s *StreamLogger) writeMessage(fnType logFunctionType, message string) {
	fnInformation := fnTypeInformationMap[fnType]
	if s.level >= fnInformation.logLevel {
		if s.format == RawFormat {
			_, _ = s.stream.Write([]byte(message))
		} else if s.format == TimeFormat {
			if env.GlobalGetEnv(DevSpaceLogTimestamps) == "true" || s.level == logrus.DebugLevel {
				now := time.Now()
				_, _ = s.stream.Write([]byte(ansi.Color(formatInt(now.Hour())+":"+formatInt(now.Minute())+":"+formatInt(now.Second())+" ", "white+b")))
			}
			_, _ = s.stream.Write([]byte(message))
		} else if s.format == TextFormat {
			if env.GlobalGetEnv(DevSpaceLogTimestamps) == "true" || s.level == logrus.DebugLevel {
				now := time.Now()
				_, _ = s.stream.Write([]byte(ansi.Color(formatInt(now.Hour())+":"+formatInt(now.Minute())+":"+formatInt(now.Second())+" ", "white+b")))
			}
			_, _ = s.stream.Write([]byte(ansi.Color(fnInformation.tag, fnInformation.color)))
			_, _ = s.stream.Write([]byte(message))
		} else if s.format == JsonFormat {
			s.writeJSON(message, fnInformation.logLevel)
		}
	}
}

func (s *StreamLogger) writeJSON(message string, level logrus.Level) {
	line, err := json.Marshal(&Line{
		Time:    time.Now(),
		Message: stripansi.Strip(strings.TrimSpace(message)),
		Level:   level,
	})
	if err == nil {
		_, _ = s.stream.Write([]byte(string(line) + "\n"))
	}
}

func (s *StreamLogger) Debug(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(debugFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Debugf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(debugFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Children() []Logger {
	return nil
}

func (s *StreamLogger) Info(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(infoFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Infof(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(infoFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Warn(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(warnFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Warnf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(warnFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Error(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(errorFn, fmt.Sprintln(args...))
}

func (s *StreamLogger) Errorf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(errorFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Fatal(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	msg := fmt.Sprintln(args...)

	s.writeMessage(fatalFn, msg)
	os.Exit(1)
}

func (s *StreamLogger) Fatalf(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	msg := fmt.Sprintf(format, args...)

	s.writeMessage(fatalFn, msg+"\n")
	os.Exit(1)
}

func (s *StreamLogger) Done(args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(doneFn, fmt.Sprintln(args...))

}

func (s *StreamLogger) Donef(format string, args ...interface{}) {
	s.m.Lock()
	defer s.m.Unlock()

	s.writeMessage(doneFn, fmt.Sprintf(format, args...)+"\n")
}

func (s *StreamLogger) Print(level logrus.Level, args ...interface{}) {
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

func (s *StreamLogger) Printf(level logrus.Level, format string, args ...interface{}) {
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

func (s *StreamLogger) SetLevel(level logrus.Level) {
	s.m.Lock()
	defer s.m.Unlock()

	s.level = level
}

func (s *StreamLogger) GetLevel() logrus.Level {
	s.m.Lock()
	defer s.m.Unlock()

	return s.level
}

func (s *StreamLogger) Writer(level logrus.Level) io.WriteCloser {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < level {
		return &NopCloser{ioutil.Discard}
	}

	return &NopCloser{s}
}

func (s *StreamLogger) Write(message []byte) (int, error) {
	s.m.Lock()
	defer s.m.Unlock()

	return s.write(message)
}

func (s *StreamLogger) WriteString(level logrus.Level, message string) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.level < level {
		return
	}

	_, _ = s.write([]byte(message))
}

func (s *StreamLogger) write(message []byte) (int, error) {
	var (
		n   int
		err error
	)
	if s.format == JsonFormat {
		s.writeJSON(string(message), logrus.InfoLevel)
		n = len(message)
	} else {
		n, err = s.stream.Write(message)
	}
	return n, err
}

func (s *StreamLogger) Question(params *survey.QuestionOptions) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.isTerminal {
		return "", fmt.Errorf("cannot ask question '%s' because you are not currently using a terminal", params.Question)
	}

	// Check if we can ask the question
	if s.level < logrus.InfoLevel {
		return "", errors.Errorf("cannot ask question '%s' because log level is too low", params.Question)
	}

	_, _ = s.write([]byte("\n"))
	return s.survey.Question(params)
}
