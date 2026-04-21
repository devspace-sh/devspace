package framework

import "os"

func Setenv(key, value string) {
	ExpectNoError(os.Setenv(key, value))
}

func Unsetenv(key string) {
	ExpectNoError(os.Unsetenv(key))
}
