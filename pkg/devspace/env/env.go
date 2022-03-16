package env

import "os"

var GlobalGetEnv = func(name string) string {
	return os.Getenv(name)
}
