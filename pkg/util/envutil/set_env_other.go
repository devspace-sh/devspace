// +build freebsd,amd64 !freebsd

package envutil

import (
	"errors"
	"runtime"

	"github.com/badgerodon/penv"
)

// This is necessary because the mitchellh/go-ps package has a bug and cannot compile on freebsd 386
func setEnv(name string, value string) error {
	if runtime.GOOS == "windows" && name == "PATH" && len(value) > 2047 {
		return errors.New("Cannot set PATH env var because value is longer than 2047 characters")
	}
	return penv.SetEnv(name, value)
}
