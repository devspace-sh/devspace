// +build windows

package analytics

import (
	"os"
	"syscall"
)

func sigterm(pid int) {
	d, e := syscall.LoadDLL("kernel32.dll")
	if e != nil {
		return
	}

	p, e := d.FindProc("GenerateConsoleCtrlEvent")
	if e != nil {
		return
	}

	r, _, _ := p.Call(uintptr(syscall.CTRL_C_EVENT), uintptr(pid))
	if r != 0 {
		os.Exit(1)
	}
}
