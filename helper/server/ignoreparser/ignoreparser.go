package ignoreparser

import (
	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"
	"strings"
)

// IgnoreParser is a wrapping interface for gitignore.IgnoreParser that
// adds a method to find out if the parser has any negating patterns
type IgnoreParser interface {
	gitignore.IgnoreParser

	// This is useful for optimization, since if an ignore parser has no
	// negate patterns, we can skip certain sub trees that do are ignored
	// by another rule.
	HasNegatePatterns() bool
}

type ignoreParser struct {
	gitignore.IgnoreParser
	NegatePatterns bool
}

func (i *ignoreParser) HasNegatePatterns() bool {
	return i.NegatePatterns
}

// CompilePaths creates a new ignore parser from a string array
func CompilePaths(excludePaths []string) (IgnoreParser, error) {
	if len(excludePaths) > 0 {
		negatePattern := false
		for _, line := range excludePaths {
			line = strings.Trim(line, " ")
			if line == "" {
				continue
			} else if line[0] == '!' {
				negatePattern = true
				break
			}
		}

		gitIgnoreParser, err := gitignore.CompileIgnoreLines(excludePaths...)
		if err != nil {
			return nil, errors.Wrap(err, "compile ignore lines")
		}

		return &ignoreParser{
			IgnoreParser:   gitIgnoreParser,
			NegatePatterns: negatePattern,
		}, nil
	}

	return nil, nil
}
