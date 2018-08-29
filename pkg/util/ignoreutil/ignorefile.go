package ignoreutil

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	glob "github.com/bmatcuk/doublestar"
)

//GetIgnoreRules reads the ignoreRules from the .dockerignore
func GetIgnoreRules(rootDirectory string) ([]string, error) {
	ignoreRules := []string{}

	ignoreFiles, err := glob.Glob(rootDirectory + "/**/.dockerignore")

	if err != nil {
		return nil, err
	}

	for _, ignoreFile := range ignoreFiles {
		ignoreBytes, err := ioutil.ReadFile(ignoreFile)

		if err != nil {
			return nil, err
		}
		pathPrefix := strings.Replace(strings.TrimPrefix(filepath.Dir(ignoreFile), rootDirectory), "\\", "/", -1)
		ignoreLines := strings.Split(string(ignoreBytes), "\r\n")

		for _, ignoreRule := range ignoreLines {
			ignoreRule = strings.Trim(ignoreRule, " ")
			initialOffset := 0

			if len(ignoreRule) > 0 && ignoreRule[initialOffset] != '#' {
				prefixedIgnoreRule := ""

				if len(pathPrefix) > 0 {
					if ignoreRule[initialOffset] == '!' {
						prefixedIgnoreRule = prefixedIgnoreRule + "!"
						initialOffset = 1
					}

					if ignoreRule[initialOffset] == '/' {
						prefixedIgnoreRule = prefixedIgnoreRule + ignoreRule[initialOffset:]
					} else {
						prefixedIgnoreRule = prefixedIgnoreRule + pathPrefix + "/**/" + ignoreRule[initialOffset:]
					}
				} else {
					prefixedIgnoreRule = ignoreRule
				}

				if prefixedIgnoreRule != "Dockerfile" && prefixedIgnoreRule != "/Dockerfile" {
					ignoreRules = append(ignoreRules, prefixedIgnoreRule)
				}
			}
		}
	}
	return ignoreRules, nil
}
