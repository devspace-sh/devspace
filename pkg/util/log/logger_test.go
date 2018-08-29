package log

/*
import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/covexo/devspace/pkg/util/fsutil"
	"github.com/stretchr/testify/assert"
)

func TestGetLogger(t *testing.T) {

	os.Remove("./.devspace/logs/TestLogger.log")

	logger := GetFileLogger("TestLogger")

	logger.Info("Some Info")
	logger.Warn("Some Warning")
	logger.Debug("Some Debug")

	f, ok := logger.Out.(*os.File)
	assert.True(t, ok)
	f.Close()

	time.Sleep(time.Second)

	fileContent, err := fsutil.ReadFile("./.devspace/logs/TestLogger.log", -1)
	assert.Nil(t, err)

	t.Logf(string(fileContent))

	logsAsStrings := strings.Split(string(fileContent), "}")
	logsAsStructs := make([]Log, len(logsAsStrings))

	for n, logAsString := range logsAsStrings {

		if n == len(logsAsStrings)-1 {
			break
		}

		json.Unmarshal([]byte(logAsString+"}"), &logsAsStructs[n])
	}

	assert.Equal(t, "info", logsAsStructs[0].Level)
	assert.Equal(t, "warning", logsAsStructs[1].Level)
	assert.Equal(t, "", logsAsStructs[2].Level)

	assert.Equal(t, "Some Info", logsAsStructs[0].Msg)
	assert.Equal(t, "Some Warning", logsAsStructs[1].Msg)
	assert.Equal(t, "", logsAsStructs[2].Msg)
}

type CustomHook struct {
}

type Log struct {
	Level string
	Msg   string
	Time  string
}
*/
