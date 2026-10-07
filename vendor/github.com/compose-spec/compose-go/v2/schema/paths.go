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

package schema

import (
	"encoding/json"
	"slices"
	"strings"
	"sync"

	"github.com/compose-spec/compose-go/v2/tree"
)

// AttributePaths returns the attribute paths ([tree.Path] patterns) the
// Compose Specification JSON schema declares, one per mapping attribute:
// pattern-keyed mappings (services, networks, per-key mounts…) contribute a
// [tree.PathMatchAll] component and sequence items a [tree.PathMatchList]
// component, matching the convention used across compose-go.
//
// It gives a runtime the complete spec-level inventory to build its
// "supported attributes" declaration from (see loader.Options
// SupportedAttributes): start from the full specification and remove the
// paths the runtime does not implement, so every newly-specified attribute is
// reported as unsupported until it is deliberately wired in.
//
// The result is sorted and stable for a given schema; the slice is shared,
// callers must not mutate it (clone before editing).
func AttributePaths() []tree.Path {
	attributePathsOnce.Do(func() {
		attributePaths = collectAttributePaths()
	})
	return attributePaths
}

var (
	attributePathsOnce sync.Once
	attributePaths     []tree.Path
)

func collectAttributePaths() []tree.Path {
	var root map[string]any
	// the embedded schema is validated by tests; a broken schema would fail
	// Validate long before this point
	if err := json.Unmarshal([]byte(Schema), &root); err != nil {
		panic(err)
	}
	defs, _ := root["$defs"].(map[string]any)
	c := &pathCollector{defs: defs}
	c.walk(root, tree.NewPath(), map[string]bool{})

	paths := make([]tree.Path, 0, len(c.paths))
	for path := range c.paths {
		paths = append(paths, tree.Path(path))
	}
	slices.Sort(paths)
	return paths
}

type pathCollector struct {
	defs  map[string]any
	paths map[string]bool
}

// walk visits a schema node and records the attribute paths its object
// properties declare. active tracks the $refs on the current branch so
// self-referencing definitions (include, service.develop…) terminate.
func (c *pathCollector) walk(node map[string]any, path tree.Path, active map[string]bool) {
	if c.paths == nil {
		c.paths = map[string]bool{}
	}
	if ref, ok := node["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/$defs/")
		if active[name] {
			return
		}
		if def, ok := c.defs[name].(map[string]any); ok {
			active[name] = true
			c.walk(def, path, active)
			delete(active, name)
		}
		return
	}
	for _, combinator := range []string{"oneOf", "anyOf", "allOf"} {
		if alternatives, ok := node[combinator].([]any); ok {
			for _, alternative := range alternatives {
				if sub, ok := alternative.(map[string]any); ok {
					c.walk(sub, path, active)
				}
			}
		}
	}
	if properties, ok := node["properties"].(map[string]any); ok {
		for name, sub := range properties {
			next := path.Next(name)
			c.paths[string(next)] = true
			if subSchema, ok := sub.(map[string]any); ok {
				c.walk(subSchema, next, active)
			}
		}
	}
	// pattern-keyed mappings (services, networks, x-* extension points…)
	// and typed additionalProperties both accept arbitrary keys: a single
	// PathMatchAll component stands for them
	for _, keyed := range []string{"patternProperties", "additionalProperties"} {
		v, ok := node[keyed].(map[string]any)
		if ok {
			if keyed == "patternProperties" {
				for patternKey, sub := range v {
					// extension escape hatches ("^x-") are not attributes of
					// the specification: recording them as a wildcard would
					// blanket-accept every sibling key
					if strings.Contains(patternKey, "x-") {
						continue
					}
					if subSchema, ok := sub.(map[string]any); ok {
						next := path.Next(tree.PathMatchAll)
						c.paths[string(next)] = true
						c.walk(subSchema, next, active)
					}
				}
			} else {
				next := path.Next(tree.PathMatchAll)
				c.paths[string(next)] = true
				c.walk(v, next, active)
			}
		}
	}
	if items, ok := node["items"].(map[string]any); ok {
		c.walk(items, path.Next(tree.PathMatchList), active)
	}
}
