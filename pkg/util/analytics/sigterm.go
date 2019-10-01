// +build !windows

package analytics

import (
	"os"
)

func sigterm(pid int) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	p.Signal(os.Interrupt)
}
