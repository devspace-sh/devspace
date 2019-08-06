// +build freebsd

package envutil

import "errors"

// This is necessary because the mitchellh/go-ps package has a bug and cannot compile on freebsd 386
func setEnv(name string, value string) error {
	return errors.New("Set Env Variables not supported on freebsd 386")
}
