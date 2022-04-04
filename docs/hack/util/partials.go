package util

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var partialNameRegex = regexp.MustCompile(`(\..*$)|[^a-zA-Z-]`)

func GetPartialImportName(partialImport string) string {
	basename := filepath.Base(partialImport)
	basename = partialNameRegex.ReplaceAllString(basename, "")

	return fmt.Sprintf("Partial%s", strings.Title(basename))
}
