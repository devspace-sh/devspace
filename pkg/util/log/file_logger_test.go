package log

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/covexo/devspace/pkg/util/fsutil"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/runtime"
)

func TestFileLoggerBasic(t *testing.T) {
	os.Remove("./.devspace/logs/TestLogger.log")
	defer os.Remove("./.devspace/logs/TestLogger.log")

	fileLogger := GetFileLogger("TestLogger")

	fileLogger.Print(logrus.InfoLevel, "TestInfo")
	fileLogger.Print(logrus.DebugLevel, "TestDebug") //Shouldn't log because of wrong loggerLevel
	fileLogger.Print(logrus.WarnLevel, "TestWarn")
	fileLogger.Print(logrus.ErrorLevel, "TestError")
	fileLogger.Done("TestDone")
	fileLogger.Fail("TestFail")

	fileLogger.Printf(logrus.InfoLevel, "TestInfof")
	fileLogger.Printf(logrus.DebugLevel, "TestDebugf")
	fileLogger.Printf(logrus.WarnLevel, "TestWarnf")
	fileLogger.Printf(logrus.ErrorLevel, "TestErrorf")
	fileLogger.Donef("TestDonef")
	fileLogger.Failf("TestFailf")

	context := make([]interface{}, 1)
	context[0] = "TestDoneContext"
	fileLogger.printWithContext(doneFn, context, "TestDone")
	context[0] = "TestFailContext"
	fileLogger.printWithContext(failFn, context, "TestFail")
	context[0] = "TestInfoContext"
	fileLogger.printWithContext(infoFn, context, "TestInfo")
	context[0] = "TestDebugContext"
	fileLogger.printWithContext(debugFn, context, "TestDebug") //Shouldn't log because of wrong loggerLevel
	context[0] = "TestWarnContext"
	fileLogger.printWithContext(warnFn, context, "TestWarn")
	context[0] = "TestErrorContext"
	fileLogger.printWithContext(errorFn, context, "TestError")

	context[0] = "TestDoneContextf"
	fileLogger.printWithContextf(doneFn, context, "TestDonef")
	context[0] = "TestFailContextf"
	fileLogger.printWithContextf(failFn, context, "TestFailf")
	context[0] = "TestInfoContextf"
	fileLogger.printWithContextf(infoFn, context, "TestInfof")
	context[0] = "TestDebugContextf"
	fileLogger.printWithContextf(debugFn, context, "TestDebugf") //Shouldn't log because of wrong loggerLevel
	context[0] = "TestWarnContextf"
	fileLogger.printWithContextf(warnFn, context, "TestWarnf")
	context[0] = "TestErrorContextf"
	fileLogger.printWithContextf(errorFn, context, "TestErrorf")

	fileLogger.Write("WrittenMessage")

	fileLogger.With("WithObject").Info("WithMessage")

	//TODO: Test those method calls
	//fileLogger.Print(logrus.FatalLevel, "TestFatal")
	//fileLogger.Printf(logrus.FatalLevel, "TestFatalf")
	//fileLogger.printWithContext(fatalFn, "TestFatalWithContext")
	//fileLogger.printWithContextf(fatalFn, "TestFatalWithContextf")

	logsAsStructs, err := GetLogs("./.devspace/logs/TestLogger.log")
	assert.Nil(t, err)

	assert.Equal(t, "info", logsAsStructs[0].Level)
	assert.Equal(t, "warning", logsAsStructs[1].Level)
	assert.Equal(t, "error", logsAsStructs[2].Level)
	assert.Equal(t, "info", logsAsStructs[3].Level)
	assert.Equal(t, "error", logsAsStructs[4].Level)
	assert.Equal(t, "info", logsAsStructs[5].Level)
	assert.Equal(t, "warning", logsAsStructs[6].Level)
	assert.Equal(t, "error", logsAsStructs[7].Level)
	assert.Equal(t, "info", logsAsStructs[8].Level)
	assert.Equal(t, "error", logsAsStructs[9].Level)
	assert.Equal(t, "info", logsAsStructs[10].Level)
	assert.Equal(t, "warning", logsAsStructs[11].Level)
	assert.Equal(t, "error", logsAsStructs[12].Level)
	assert.Equal(t, "info", logsAsStructs[13].Level)
	assert.Equal(t, "warning", logsAsStructs[14].Level)
	assert.Equal(t, "error", logsAsStructs[15].Level)
	assert.Equal(t, "info", logsAsStructs[16].Level)
	assert.Equal(t, "info", logsAsStructs[17].Level)

	assert.Equal(t, "TestInfo", logsAsStructs[0].Msg)
	assert.Equal(t, "TestWarn", logsAsStructs[1].Msg)
	assert.Equal(t, "TestError", logsAsStructs[2].Msg)
	assert.Equal(t, "TestDone", logsAsStructs[3].Msg)
	assert.Equal(t, "TestFail", logsAsStructs[4].Msg)
	assert.Equal(t, "TestInfof", logsAsStructs[5].Msg)
	assert.Equal(t, "TestWarnf", logsAsStructs[6].Msg)
	assert.Equal(t, "TestErrorf", logsAsStructs[7].Msg)
	assert.Equal(t, "TestDonef", logsAsStructs[8].Msg)
	assert.Equal(t, "TestFailf", logsAsStructs[9].Msg)
	assert.Equal(t, "TestInfo", logsAsStructs[10].Msg)
	assert.Equal(t, "TestWarn", logsAsStructs[11].Msg)
	assert.Equal(t, "TestError", logsAsStructs[12].Msg)
	assert.Equal(t, "TestInfof", logsAsStructs[13].Msg)
	assert.Equal(t, "TestWarnf", logsAsStructs[14].Msg)
	assert.Equal(t, "TestErrorf", logsAsStructs[15].Msg)
	assert.Equal(t, "WrittenMessage", logsAsStructs[16].Msg)
	assert.Equal(t, "WithMessage", logsAsStructs[17].Msg)

	assert.Equal(t, "TestInfoContext", logsAsStructs[10].Context0)
	assert.Equal(t, "TestWarnContext", logsAsStructs[11].Context0)
	assert.Equal(t, "TestErrorContext", logsAsStructs[12].Context0)
	assert.Equal(t, "TestInfoContextf", logsAsStructs[13].Context0)
	assert.Equal(t, "TestWarnContextf", logsAsStructs[14].Context0)
	assert.Equal(t, "TestErrorContextf", logsAsStructs[15].Context0)
	assert.Equal(t, "WithObject", logsAsStructs[17].Context0)

}

func TestPanic(t *testing.T) {
	os.Remove("./.devspace/logs/PanicLogger.log")

	defer func() {
		recover()

		logs, err := GetLogs("./.devspace/logs/PanicLogger.log")
		assert.Nil(t, err)

		assert.Equal(t, "panic", logs[0].Level)
		assert.Equal(t, "TestPanic", logs[0].Msg)

		os.Remove("./.devspace/logs/PanicLogger.log")
	}()

	fileLogger := GetFileLogger("PanicLogger")
	fileLogger.Print(logrus.PanicLevel, "TestPanic")

	t.Error("No Panic")
}

func TestPanicf(t *testing.T) {
	os.Remove("./.devspace/logs/PanicfLogger.log")

	defer func() {
		recover()

		logs, err := GetLogs("./.devspace/logs/PanicfLogger.log")
		assert.Nil(t, err)

		assert.Equal(t, "panic", logs[0].Level)
		assert.Equal(t, "TestPanicf", logs[0].Msg)

		os.Remove("./.devspace/logs/PanicfLogger.log")
	}()

	fileLogger := GetFileLogger("PanicfLogger")
	fileLogger.Printf(logrus.PanicLevel, "TestPanicf")

	t.Error("No Panic")
}

func TestPrintWithContextFnTypePanic(t *testing.T) {
	os.Remove("./.devspace/logs/PanicContextLogger.log")

	defer func() {
		recover()

		logs, err := GetLogs("./.devspace/logs/PanicContextLogger.log")
		assert.Nil(t, err)

		assert.Equal(t, "panic", logs[0].Level)
		assert.Equal(t, "TestPanic", logs[0].Msg)
		assert.Equal(t, "TestPanicContext", logs[0].Context0)

		os.Remove("./.devspace/logs/PanicContextLogger.log")
	}()

	context := make([]interface{}, 1)
	context[0] = "TestPanicContext"

	fileLogger := GetFileLogger("PanicContextLogger")
	fileLogger.printWithContext(panicFn, context, "TestPanic")

	t.Error("No Panic")
}

func TestPrintWithContextfFnTypePanic(t *testing.T) {
	os.Remove("./.devspace/logs/PanicContextfLogger.log")

	defer func() {
		recover()

		logs, err := GetLogs("./.devspace/logs/PanicContextfLogger.log")
		assert.Nil(t, err)

		assert.Equal(t, "panic", logs[0].Level)
		assert.Equal(t, "TestPanicf", logs[0].Msg)
		assert.Equal(t, "TestPanicContextf", logs[0].Context0)

		os.Remove("./.devspace/logs/PanicContextfLogger.log")
	}()

	context := make([]interface{}, 1)
	context[0] = "TestPanicContextf"

	fileLogger := GetFileLogger("PanicContextfLogger")
	fileLogger.printWithContextf(panicFn, context, "TestPanicf")

	t.Error("No Panic")
}

func TestOverrideRuntimeErrorHandler(t *testing.T) {
	os.Remove("./.devspace/logs/errors.log")
	defer os.Remove("./.devspace/logs/errors.log")

	OverrideRuntimeErrorHandler()

	err := errors.New("TestErr")
	runtime.HandleError(err)
}

func GetLogs(path string) ([]Log, error) {

	fileContent, err := fsutil.ReadFile(path, -1)
	if err != nil {
		return nil, err
	}

	logsAsStrings := strings.Split(string(fileContent), "}")
	logsAsStructs := make([]Log, len(logsAsStrings))

	for n, logAsString := range logsAsStrings {
		if n == len(logsAsStrings)-1 {
			break
		}
		json.Unmarshal([]byte(logAsString+"}"), &logsAsStructs[n])
	}

	return logsAsStructs, nil

}

type Log struct {
	Level    string
	Msg      string
	Time     string
	Context0 string `json:"context-0"`
}
