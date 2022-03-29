package ignoreparser

import (
	"github.com/loft-sh/devspace/pkg/util/log"
	"path"
	"strings"

	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"
)

// IgnoreParser is a wrapping interface for gitignore.IgnoreParser that
// adds a method to find out if the parser has any negating patterns
type IgnoreParser interface {
	// Matches returns if the given relative path matches the ignore parser
	Matches(relativePath string, isDir bool) bool

	// RequireFullScan is useful for optimization, since if an ignore parser has no
	// general negate patterns, we can skip certain sub trees that do are ignored
	// by another rule.
	RequireFullScan() bool
}

type ignoreParser struct {
	ignoreParser gitignore.IgnoreParser

	absoluteNegatePatterns []string
	requireFullScan        bool
}

func (i *ignoreParser) Matches(relativePath string, isDir bool) bool {
	relativePath = strings.TrimRight(relativePath, "/")
	if isDir {
		relativePath = relativePath + "/"
	}

	if strings.HasPrefix(relativePath, "./") {
		relativePath = relativePath[1:]
	} else if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}

	if isDir {
		for _, p := range i.absoluteNegatePatterns {
			if strings.Index(p, relativePath) == 0 {
				return false
			}
		}
	}

	return i.ignoreParser.MatchesPath(relativePath)
}

func (i *ignoreParser) RequireFullScan() bool {
	return i.requireFullScan
}

// CompilePaths creates a new ignore parser from a string array
func CompilePaths(excludePaths []string, log log.Logger) (IgnoreParser, error) {
	if len(excludePaths) > 0 {
		requireFullScan := false
		absoluteNegatePatterns := []string{}
		for _, line := range excludePaths {
			line = strings.Trim(line, " ")
			if line == "" {
				continue
			} else if line[0] == '!' {
				if len(line) > 1 && line[1] == '/' {
					p := line[1:]
					if !strings.Contains(p, "**") && !strings.Contains(path.Dir(p), "*") {
						absoluteNegatePatterns = append(absoluteNegatePatterns, p)
					} else {
						log.Warnf("Exclude path '%s' uses a ** or * and thus requires a full initial scan. Please consider using a path in the form of '!/path/to/my/folder/' instead to improve performance", line)
						requireFullScan = true
					}
				} else {
					log.Warnf("Exclude path '%s' is not scoped to the directory base and thus requires a full initial scan. Please consider using a path in the form of '!/path/to/my/folder/' instead to improve performance", line)
					requireFullScan = true
				}
			}
		}

		gitIgnoreParser, err := gitignore.CompileIgnoreLines(excludePaths...)
		if err != nil {
			return nil, errors.Wrap(err, "compile ignore lines")
		}

		return &ignoreParser{
			ignoreParser:           gitIgnoreParser,
			absoluteNegatePatterns: absoluteNegatePatterns,
			requireFullScan:        requireFullScan,
		}, nil
	}

	return nil, nil
}
