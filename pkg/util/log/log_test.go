package log

import (
	"bufio"
	"os"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daviddengcn/go-colortext"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
)

func TestStdoutLoggerBasic(t *testing.T) {

	readers, err := setOutputStreams()
	assert.Nil(t, err)

	SetLevel(logrus.DebugLevel)
	StartFileLogging()
	Print(logrus.InfoLevel, "TestPrintInfo")
	Print(logrus.DebugLevel, "TestPrintDebug")
	Print(logrus.WarnLevel, "TestPrintWarn")
	Print(logrus.ErrorLevel, "TestPrintError")

	Debug("TestDebug")
	Info("TestInfo")
	Warn("TestWarn")
	Error("TestError")
	Done("TestDone")
	Fail("TestFail")

	Printf(logrus.InfoLevel, "TestPrintInfof")
	Printf(logrus.DebugLevel, "TestPrintDebugf")
	Printf(logrus.WarnLevel, "TestPrintWarnf")
	Printf(logrus.ErrorLevel, "TestPrintErrorf")

	Debugf("TestDebugf")
	Infof("TestInfof")
	Warnf("TestWarnf")
	Errorf("TestErrorf")
	Donef("TestDonef")
	Failf("TestFailf")

	Write("TestWrite")
	context := make([]interface{}, 1)
	context[0] = "TestContext"
	GetInstance().printWithContextf(infoFn, context, "TestWithContextf")

	With("SomeContext").Print(logrus.DebugLevel, "TestWithDebug")
	With("SomeContext").Print(logrus.InfoLevel, "TestWithInfo")
	With("SomeContext").Print(logrus.WarnLevel, "TestWithWarn")
	With("SomeContext").Print(logrus.ErrorLevel, "TestWithError")
	With("SomeContext").Done("TestWithDone")
	With("SomeContext").Fail("TestWithFail")

	With("SomeContext").Printf(logrus.DebugLevel, "TestWithDebugf")
	With("SomeContext").Printf(logrus.InfoLevel, "TestWithInfof")
	With("SomeContext").Printf(logrus.WarnLevel, "TestWithWarnf")
	With("SomeContext").Printf(logrus.ErrorLevel, "TestWithErrorf")
	With("SomeContext").Donef("TestWithDonef")
	With("SomeContext").Failf("TestWithFailf")

	With("SomeContext").With("MoreContext").Info("TestWithWithInfo")

	//TODO: Find a way to get the color of the output
	//WriteColored("TestWriteColored", ct.Magenta)

	expectedDebug := "[DEBUG]  TestPrintDebug\n" +
		"[DEBUG]  TestDebug\n" +
		"[DEBUG]  TestPrintDebugf\n" +
		"[DEBUG]  TestDebugf\n"
	buf := make([]byte, len(expectedDebug))
	_, err = readers[0].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedDebug, string(buf))

	expectedInfo := "[INFO]   TestPrintInfo\n" +
		"[INFO]   TestInfo\n" +
		"[INFO]   TestPrintInfof\n" +
		"[INFO]   TestInfof\n" +
		"TestWrite" +
		"[INFO]   TestWithContextf\n" +
		"[INFO]   TestWithInfo\n" +
		"[INFO]   TestWithInfof\n" +
		"[INFO]   TestWithWithInfo\n"
	buf = make([]byte, len(expectedInfo))
	_, err = readers[1].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedInfo, string(buf))

	expectedWarn := "[WARN]   TestPrintWarn\n" +
		"[WARN]   TestWarn\n" +
		"[WARN]   TestPrintWarnf\n" +
		"[WARN]   TestWarnf\n" +
		"[WARN]   TestWithWarn\n" +
		"[WARN]   TestWithWarnf\n"
	buf = make([]byte, len(expectedWarn))
	_, err = readers[2].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWarn, string(buf))

	expectedError := "[ERROR]  TestPrintError\n" +
		"[ERROR]  TestError\n" +
		"[ERROR]  TestPrintErrorf\n" +
		"[ERROR]  TestErrorf\n" +
		"[ERROR]  TestWithError\n" +
		"[ERROR]  TestWithErrorf\n"
	buf = make([]byte, len(expectedError))
	_, err = readers[3].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedError, string(buf))

	expectedDone := "[DONE] √ TestDone\n" +
		"[DONE] √ TestDonef\n" +
		"[DONE] √ TestWithDone\n" +
		"[DONE] √ TestWithDonef\n"
	buf = make([]byte, len(expectedDone))
	_, err = readers[6].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedDone, string(buf))

	expectedFail := "[FAIL] X TestFail\n" +
		"[FAIL] X TestFailf\n" +
		"[FAIL] X TestWithFail\n" +
		"[FAIL] X TestWithFailf\n"
	buf = make([]byte, len(expectedFail))
	_, err = readers[7].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedFail, string(buf))

	osStdout := GetInstance().GetStream()
	assert.Equal(t, os.Stdout, osStdout)

}

func TestLogPanic(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	StartFileLogging()
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Panic("TestPanic")
	t.Error("No Panic")
}

func TestLogPanicf(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	StartFileLogging()
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPanicf\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Panicf("TestPanicf")
	t.Error("No Panic")
}

func TestLogPrintPanic(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	StartFileLogging()
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPrintPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Print(logrus.PanicLevel, "TestPrintPanic")
	t.Error("No Panic")
}

func TestLogPrintfPanic(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	StartFileLogging()
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPrintfPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Printf(logrus.PanicLevel, "TestPrintfPanic")
	t.Error("No Panic")
}

func TestLogPanicNoFileLogger(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	stdoutLog.fileLogger = nil
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Panic("TestPanic")
	t.Error("No Panic")
}

func TestLogPanicfNoFileLogger(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	stdoutLog.fileLogger = nil
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPanicf\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Panicf("TestPanicf")
	t.Error("No Panic")
}

func TestLogPrintPanicNoFileLogger(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	stdoutLog.fileLogger = nil
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPrintPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Print(logrus.PanicLevel, "TestPrintPanic")
	t.Error("No Panic")
}

func TestLogPrintfPanicNoFileLogger(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	stdoutLog.fileLogger = nil
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestPrintfPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	Printf(logrus.PanicLevel, "TestPrintfPanic")
	t.Error("No Panic")
}

func TestLoggerPrintPanic(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	StartFileLogging()
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestWithPrintPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	With("Context").Print(logrus.PanicLevel, "TestWithPrintPanic")
	t.Error("No Panic")
}

func TestLoggerPrintfPanic(t *testing.T) {
	readers, err := setOutputStreams()
	assert.Nil(t, err)
	StartFileLogging()
	defer func() {
		recover()

		expectedPanic := "[PANIC]  TestWithPrintfPanic\n"
		buf := make([]byte, len(expectedPanic))
		_, err := readers[5].Read(buf)
		assert.Nil(t, err)
		assert.Equal(t, expectedPanic, string(buf))

	}()
	With("Context").Printf(logrus.PanicLevel, "TestWithPrintfPanic")
	t.Error("No Panic")
}

func TestPrintTable(t *testing.T) {

	header := make([]string, 1)
	values := make([][]string, 2)
	values[0] = make([]string, 1)
	values[1] = make([]string, 1)

	header[0] = "head"
	values[0][0] = "value"
	values[1][0] = "longer_value"

	readers, err := setOutputStreams()
	assert.Nil(t, err)

	PrintTable(header, values)

	expectedTable := " head          \n" +
		" value         \n" +
		" longer_value  \n"

	buf := make([]byte, len(expectedTable))
	_, err = readers[1].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedTable, string(buf))
}

func TestWaitFeature(t *testing.T) {

	r, w, err := os.Pipe()
	assert.Nil(t, err)

	oldStdout := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()
	StartWait("TestWait")

	time.Sleep(waitInterval / 5)

	expectedWait := "[WAIT] | TestWait"
	buf := make([]byte, len(expectedWait))
	_, err = r.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWait, string(buf))

	time.Sleep(waitInterval)

	expectedWait = "\r[WAIT] / TestWait"
	buf = make([]byte, len(expectedWait))
	_, err = r.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWait, string(buf))

	readers, err := setOutputStreams()
	assert.Nil(t, err)
	Write("TestWriteBetweenWaits")
	expectedWrite := "TestWriteBetweenWaits"
	buf = make([]byte, len(expectedWrite))
	_, err = readers[1].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWrite, string(buf))

	time.Sleep(waitInterval / 5)

	expectedWait = "\r                 \r[WAIT] - TestWait"
	buf = make([]byte, len(expectedWait))
	_, err = r.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWait, string(buf))

	Info("TestInfoBetweenWaits")
	expectedInfo := "[INFO]   TestInfoBetweenWaits\n"
	buf = make([]byte, len(expectedInfo))
	_, err = readers[1].Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedInfo, string(buf))

	time.Sleep(waitInterval / 5)

	expectedWait = "\r                 \r[WAIT] \\ TestWait"
	buf = make([]byte, len(expectedWait))
	_, err = r.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWait, string(buf))

	StartWait("TestWait2")

	time.Sleep(waitInterval / 5)

	expectedWait = "\r                 \r[WAIT] | TestWait2"
	buf = make([]byte, len(expectedWait))
	_, err = r.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWait, string(buf))

	StopWait()
	time.Sleep(waitInterval / 10)

	expectedWait = "\r                  \r"
	buf = make([]byte, len(expectedWait))
	_, err = r.Read(buf)
	assert.Nil(t, err)
	assert.Equal(t, expectedWait, string(buf))

}

func setOutputStreams() ([]*bufio.Reader, error) {
	debugReader, debugWriter, debugErr := os.Pipe()
	infoReader, infoWriter, infoErr := os.Pipe()
	warnReader, warnWriter, warnErr := os.Pipe()
	errorReader, errorWriter, errorErr := os.Pipe()
	fatalReader, fatalWriter, fatalErr := os.Pipe()
	panicReader, panicWriter, panicErr := os.Pipe()
	doneReader, doneWriter, doneErr := os.Pipe()
	failReader, failWriter, failErr := os.Pipe()

	if debugErr != nil || infoErr != nil || warnErr != nil || errorErr != nil || fatalErr != nil || panicErr != nil || doneErr != nil || failErr != nil {
		return nil, errors.New("Error ocurred while creating streams")
	}

	fnTypeInformationMap = map[logFunctionType]*fnTypeInformation{
		debugFn: {
			tag:      "[DEBUG]  ",
			color:    ct.Green,
			logLevel: logrus.DebugLevel,
			stream:   debugWriter,
		},
		infoFn: {
			tag:      "[INFO]   ",
			color:    ct.Green,
			logLevel: logrus.InfoLevel,
			stream:   infoWriter,
		},
		warnFn: {
			tag:      "[WARN]   ",
			color:    ct.Red,
			logLevel: logrus.WarnLevel,
			stream:   warnWriter,
		},
		errorFn: {
			tag:      "[ERROR]  ",
			color:    ct.Red,
			logLevel: logrus.ErrorLevel,
			stream:   errorWriter,
		},
		fatalFn: {
			tag:      "[FATAL]  ",
			color:    ct.Red,
			logLevel: logrus.FatalLevel,
			stream:   fatalWriter,
		},
		panicFn: {
			tag:      "[PANIC]  ",
			color:    ct.Red,
			logLevel: logrus.PanicLevel,
			stream:   panicWriter,
		},
		doneFn: {
			tag:      "[DONE] √ ",
			color:    ct.Green,
			logLevel: logrus.InfoLevel,
			stream:   doneWriter,
		},
		failFn: {
			tag:      "[FAIL] X ",
			color:    ct.Red,
			logLevel: logrus.ErrorLevel,
			stream:   failWriter,
		},
	}

	readerArray := []*bufio.Reader{
		bufio.NewReader(debugReader),
		bufio.NewReader(infoReader),
		bufio.NewReader(warnReader),
		bufio.NewReader(errorReader),
		bufio.NewReader(fatalReader),
		bufio.NewReader(panicReader),
		bufio.NewReader(doneReader),
		bufio.NewReader(failReader),
	}

	return readerArray, nil
}
