package envutil

import (
	"github.com/badgerodon/penv"
)

func SetEnvVar(name string, value string) error {
	return penv.SetEnv(name, value)
}
