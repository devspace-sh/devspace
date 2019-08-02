// +build windows

package envutil

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	SHCNE_ASSOCCHANGED = 0x08000000
	SHCNF_IDLIST       = 0x0000

	HWND_BROADCAST   = 0xFFFF
	WM_SETTINGCHANGE = 0x001A
	SMTO_ABORTIFHUNG = 0x0002
)

// This is necessary because the mitchellh/go-ps package has a bug and cannot compile on freebsd 386
func setEnv(name string, value string) error {
	fmt.Println(name)
	fmt.Println(value)
	k, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.ALL_ACCESS)
	if err != nil {
		log.Fatal(err)
	}
	defer k.Close()

	err = k.SetExpandStringValue(name, value)
	if err != nil {
		log.Fatal(err)
	}
	// https://docs.microsoft.com/en-us/windows/desktop/api/shlobj_core/nf-shlobj_core-shchangenotify
	syscall.NewLazyDLL("shell32.dll").NewProc("SHChangeNotify").Call(
		uintptr(SHCNE_ASSOCCHANGED),
		uintptr(SHCNF_IDLIST),
		0, 0)

	// https://docs.microsoft.com/en-us/windows/desktop/api/winuser/nf-winuser-sendmessagetimeoutw
	env, _ := syscall.UTF16PtrFromString("Environment")
	syscall.NewLazyDLL("user32.dll").NewProc("SendMessageTimeoutW").Call(
		uintptr(HWND_BROADCAST),
		uintptr(WM_SETTINGCHANGE),
		0,
		uintptr(unsafe.Pointer(env)),
		uintptr(SMTO_ABORTIFHUNG),
		uintptr(5000))

	return nil
}
