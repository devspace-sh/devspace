// +build !windows

package analytics

import (
	"os"
)

func sigterm(pid int) {
	p, err := os.FindProcess(pid)
	if err == nil {
		p.Signal(os.Interrupt)
	}
}