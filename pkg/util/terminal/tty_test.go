package terminal

import (
	"bytes"
	"os"
	//"strings"
	"testing"

	//"github.com/docker/docker/pkg/term"
)

func TestTTY(t *testing.T) {
	//reader := strings.NewReader("")
	buf := make([]byte, 1000)
	writer := bytes.NewBuffer(buf)

	SetupTTY(os.Stdin, writer)
}
