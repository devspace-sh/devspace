package terminal

import (
	"bytes"
	"os"
	//"strings"
	"testing"

	//"github.com/docker/docker/pkg/term"
	
	"gotest.tools/assert"
)

func TestTTY(t *testing.T) {
	//reader := strings.NewReader("")
	buf := make([]byte, 1000)
	writer := bytes.NewBuffer(buf)

	tty := SetupTTY(os.Stdin, writer)
	assert.Equal(t, false, tty.Raw, "Raw terminal that doesn't got a terminal stdin")
}
