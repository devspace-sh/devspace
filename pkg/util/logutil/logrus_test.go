package logutil

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/covexo/devspace/pkg/util/fsutil"
)

func TestGetLogger(t *testing.T) {

	fsutil.WriteToFile([]byte(""), "./.devspace/logs/TestLogger.log")
	
	logger := GetLogger("TestLogger", true)
	
	logger.Info("Some Info")
	logger.Warn("Some Warning")
	logger.Debug("Some Debug")

	fileContent, err := fsutil.ReadFile("./.devspace/logs/TestLogger.log", -1)
	assert.Nil(t, err)

	t.Logf(string(fileContent))

	logsAsStrings := strings.Split(string(fileContent), "}")
	logsAsStructs := make([]Log, len(logsAsStrings))

	for n, logAsString := range logsAsStrings {

		if n == len(logsAsStrings)-1 {
			break
		}

		json.Unmarshal([]byte(logAsString + "}"), &logsAsStructs[n])
	}

	assert.Equal(t, "info", logsAsStructs[0].Level)
	assert.Equal(t, "warning", logsAsStructs[1].Level)
	assert.Equal(t, "debug", logsAsStructs[2].Level)

	assert.Equal(t, "Some Info", logsAsStructs[0].Msg)
	assert.Equal(t, "Some Warning", logsAsStructs[1].Msg)
	assert.Equal(t, "Some Debug", logsAsStructs[2].Msg)
}


type Log struct {
	Level string
	Msg string
	Time string 
}
