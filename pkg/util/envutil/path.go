package envutil

import (
	"os"
	"runtime"
	"strings"
)

//AddToPath adds a folder path to the environment variable "PATH"
func AddToPath(path string) error {
	envVarPath := "PATH"
	pathSeparator := ":"

	if runtime.GOOS == "windows" {
		pathSeparator = ";"
	}
	pathVar := os.Getenv(envVarPath)
	paths := strings.Split(pathVar, pathSeparator)
	pathIsPresent := false

	for _, existingPath := range paths {
		if path == existingPath {
			pathIsPresent = true
			break
		}
	}

	if !pathIsPresent {
		paths = append(paths, path)
	}
	return SetEnvVar(envVarPath, strings.Join(paths, pathSeparator))
}
