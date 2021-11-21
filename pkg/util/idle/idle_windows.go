//go:build windows
// +build windows

package idle

import (
	"errors"
	"syscall"
	"time"
	"unsafe"
)

// NewIdleGetter returns a new idle getter for windows
func NewIdleGetter() (Getter, error) {
	user32, err := syscall.LoadDLL("user32.dll")
	if err != nil {
		return nil, err
	}

	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return nil, err
	}

	getLastInputInfo, err := user32.FindProc("GetLastInputInfo")
	if err != nil {
		return nil, err
	}

	getTickCount, err := kernel32.FindProc("GetTickCount")
	if err != nil {
		return nil, err
	}

	return &idleGetter{
		getLastInputInfo: getLastInputInfo,
		getTickCount:     getTickCount,
	}, nil
}

type idleGetter struct {
	getLastInputInfo *syscall.Proc
	getTickCount     *syscall.Proc

	lastInputInfo struct {
		cbSize uint32
		dwTime uint32
	}
}

func (i *idleGetter) Idle() (time.Duration, error) {
	i.lastInputInfo.cbSize = uint32(unsafe.Sizeof(i.lastInputInfo))
	currentTickCount, _, _ := i.getTickCount.Call()
	r1, _, err := i.getLastInputInfo.Call(uintptr(unsafe.Pointer(&i.lastInputInfo)))
	if r1 == 0 {
		return 0, errors.New("error getting last input info: " + err.Error())
	}

	return time.Duration((uint32(currentTickCount) - i.lastInputInfo.dwTime)) * time.Millisecond, nil
}
