/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package tree

// Matcher matches concrete attribute paths against a set of declared path
// patterns. Patterns use the same tokens as [Path.Matches]: [PathMatchAll]
// for any mapping key and [PathMatchList] for sequence items.
//
// Matching is exact by construction — declaring a path accepts that node
// only, not its subtree — so removing one leaf from a declared set reliably
// surfaces it. [Matcher.MayContain] tells a caller walking a nested
// structure when descending can still reach a declared node.
type Matcher struct {
	patterns [][]string
}

// NewMatcher returns a Matcher for the given path patterns.
func NewMatcher(patterns ...Path) *Matcher {
	m := &Matcher{patterns: make([][]string, 0, len(patterns))}
	for _, pattern := range patterns {
		m.patterns = append(m.patterns, pattern.Parts())
	}
	return m
}

// Matches reports whether path matches one of the declared patterns.
func (m *Matcher) Matches(path Path) bool {
	parts := path.Parts()
	for _, pattern := range m.patterns {
		if len(pattern) == len(parts) && matchParts(pattern, parts) {
			return true
		}
	}
	return false
}

// MayContain reports whether path is a strict ancestor of at least one
// pattern: even when the node itself is not declared, a declared attribute
// lives somewhere underneath it.
func (m *Matcher) MayContain(path Path) bool {
	parts := path.Parts()
	for _, pattern := range m.patterns {
		if len(pattern) > len(parts) && matchParts(pattern[:len(parts)], parts) {
			return true
		}
	}
	return false
}

func matchParts(pattern, parts []string) bool {
	for i, part := range parts {
		switch pattern[i] {
		case PathMatchAll, part:
			continue
		default:
			return false
		}
	}
	return true
}
