//go:build darwin
// +build darwin

package idle

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NewIdleGetter returns a new idle getter for windows
func NewIdleGetter() (Getter, error) {
	return &idleGetter{}, nil
}

type idleGetter struct{}

func (i *idleGetter) fetch() ([]byte, error) {
	return exec.Command("ioreg", "-c", "IOHIDSystem").Output()
}

func (i *idleGetter) Idle() (time.Duration, error) {
	var (
		output   time.Duration
		idleInNs string
	)

	ioRegOutput, err := i.fetch()
	if err != nil {
		return output, err
	}

	rawStr := string(ioRegOutput)
	lines := strings.Split(rawStr, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "HIDIdleTime") {
			continue
		}

		cols := strings.Split(line, " ")
		idleInNs = cols[len(cols)-1]
		break
	}
	if idleInNs == "" {
		return 0, fmt.Errorf("idle time couldn't be found")
	}

	return time.ParseDuration(fmt.Sprintf("%sns", idleInNs))
}
