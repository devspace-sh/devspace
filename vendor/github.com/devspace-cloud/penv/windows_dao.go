//+build windows

package penv

import (
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/lxn/win"

	"golang.org/x/sys/windows/registry"
)

const (
	SMTO_ABORTIFHUNG uint32 = 0x0002
)

var (
	libuser32              *syscall.DLL
	sendMessageTimeoutAddr *syscall.Proc
)

func init() {
	libuser32 = syscall.MustLoadDLL("user32.dll")
	sendMessageTimeoutAddr = libuser32.MustFindProc("SendMessageTimeoutW")
}

func sendMessageTimeout(hWnd win.HWND, msg uint32, wParam, lParam uintptr, fuFlags, uTimeout uint32, lpdwResult uintptr) uintptr {
	r1, _, _ := sendMessageTimeoutAddr.Call(
		uintptr(hWnd),
		uintptr(msg),
		wParam,
		lParam,
		uintptr(fuFlags),
		uintptr(uTimeout),
		lpdwResult)

	return r1
}

// WindowsDAO is the data access object for windows
type WindowsDAO struct {
}

func init() {
	RegisterDAO(1000, func() bool {
		return runtime.GOOS == "windows"
	}, &WindowsDAO{})
}

// Load loads the environment
func (dao *WindowsDAO) Load() (*Environment, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer key.Close()

	ki, err := key.Stat()
	if err != nil {
		return nil, err
	}

	names, err := key.ReadValueNames(int(ki.ValueCount))
	if err != nil {
		return nil, err
	}

	env := &Environment{
		Setters:   make([]NameValue, 0),
		Appenders: make([]NameValue, 0),
		Unsetters: make([]NameValue, 0),
	}

	for _, name := range names {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			return nil, err
		}
		env.Setters = append(env.Setters, NameValue{strings.ToUpper(name), value})
	}

	return env, nil
}

// Save saves the environment
func (dao *WindowsDAO) Save(env *Environment) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer key.Close()

	ki, err := key.Stat()
	if err != nil {
		return err
	}

	names, err := key.ReadValueNames(int(ki.ValueCount))
	if err != nil {
		return err
	}

	for i := range env.Setters {
		env.Setters[i].Value = strings.Replace(env.Setters[i].Value, "/", "\\", -1)
	}
	for i := range env.Appenders {
		env.Appenders[i].Value = strings.Replace(env.Appenders[i].Value, "/", "\\", -1)
	}

	// set
set_loop:
	for _, nv := range env.Setters {
		for _, name := range names {
			if strings.ToUpper(name) == strings.ToUpper(nv.Name) {
				value, _, err := key.GetStringValue(name)
				if err != nil {
					return err
				}
				if value == nv.Value {
					continue set_loop
				}
			}
		}
		err = key.SetExpandStringValue(nv.Name, nv.Value)
		if err != nil {
			return err
		}
	}

	// append
append_loop:
	for _, nv := range env.Appenders {
		values := []string{}
		for _, name := range names {
			if strings.ToUpper(name) == strings.ToUpper(nv.Name) {
				value, _, err := key.GetStringValue(name)
				if err != nil {
					return err
				}
				values = append(values, strings.Split(value, ";")...)
				values = uniquei(append(values, nv.Value))
				err = key.SetExpandStringValue(nv.Name, strings.Join(values, ";"))
				if err != nil {
					return err
				}
				break append_loop
			}
		}
		err = key.SetExpandStringValue(nv.Name, nv.Value)
		if err != nil {
			return err
		}
	}

	// unset
	for _, name := range names {
		for _, nv := range env.Unsetters {
			if nv.Name == name {
				err = key.DeleteValue(name)
				if err != nil {
					return err
				}
			}
		}
	}

	str := "Environment"
	pstr, _ := syscall.UTF16PtrFromString(str)
	sendMessageTimeout(win.HWND_BROADCAST, win.WM_WININICHANGE, 0, uintptr(unsafe.Pointer(pstr)), SMTO_ABORTIFHUNG, 100, 0)

	return nil
}
