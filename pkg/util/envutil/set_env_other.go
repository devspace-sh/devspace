// +build freebsd,amd64 !freebsd

package envutil

import (
	"github.com/badgerodon/penv"
)

// This is necessary because the mitchellh/go-ps package has a bug and cannot compile on freebsd 386
func setEnv(name string, value string) error {
	return penv.SetEnv(name, value)
}
