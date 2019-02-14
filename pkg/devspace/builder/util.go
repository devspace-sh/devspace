package builder

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// CreateTempDockerfile creates a new temporary dockerfile that appends a new entrypoint and cmd
func CreateTempDockerfile(dockerfile string, entrypointArr []*string) (string, error) {
	if entrypointArr == nil || len(entrypointArr) == 0 {
		return "", errors.New("Entrypoint is empty")
	}

	// Convert to string array
	entrypoint := []string{}
	for _, str := range entrypointArr {
		if str != nil {
			entrypoint = append(entrypoint, *str)
		}
	}
	if len(entrypoint) == 0 {
		return "", errors.New("Entrypoint is empty")
	}

	data, err := ioutil.ReadFile(dockerfile)
	if err != nil {
		return "", err
	}

	// Overwrite entrypoint and cmd
	newDockerfileContents := string(data)
	newDockerfileContents += "\n\nENTRYPOINT [\"" + entrypoint[0] + "\"]"
	newDockerfileContents += "\nCMD [\"" + strings.Join(entrypoint[1:], "\",\"") + "\"]"

	tmpDir, err := ioutil.TempDir("", "example")
	if err != nil {
		return "", err
	}

	tmpfn := filepath.Join(tmpDir, "Dockerfile")
	if err := ioutil.WriteFile(tmpfn, []byte(newDockerfileContents), 0666); err != nil {
		return "", err
	}

	return tmpfn, nil
}
