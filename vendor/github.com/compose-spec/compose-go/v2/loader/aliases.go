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

package loader

import (
	"strconv"

	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/tree"
)

// extensionAlias declares that the specification attribute `to` was used as
// the x-* extension `from` before being adopted.
type extensionAlias struct {
	// parent is the path of the mapping holding the attribute.
	parent tree.Path
	from   string
	to     string
}

// extensionAliases is ordered from the outermost to the innermost attribute.
var extensionAliases = []extensionAlias{
	{parent: "services.*", from: "x-develop", to: "develop"},
	{parent: "services.*.develop.watch.[]", from: "x-initialSync", to: "initial_sync"},
}

var aliasParents = func() *tree.Matcher {
	parents := make([]tree.Path, len(extensionAliases))
	for i, alias := range extensionAliases {
		parents[i] = alias.parent
	}
	return tree.NewMatcher(parents...)
}()

// promoteAliases renames in place the extensions declared in extensionAliases
// to the attribute they stand for. When the mapping already sets a value for
// the attribute, the extension is merged under it, so that the attribute wins.
func promoteAliases(value any, p tree.Path) error {
	switch v := value.(type) {
	case map[string]any:
		for _, alias := range extensionAliases {
			if !p.Matches(alias.parent) {
				continue
			}
			ext, ok := v[alias.from]
			if !ok {
				continue
			}
			delete(v, alias.from)
			if v[alias.to] == nil {
				v[alias.to] = ext
				continue
			}
			merged, err := override.MergeYaml(deepClone(ext), v[alias.to], p.Next(alias.to))
			if err != nil {
				return err
			}
			v[alias.to] = merged
		}
		for key, e := range v {
			next := p.Next(key)
			if aliasParents.Matches(next) || aliasParents.MayContain(next) {
				if err := promoteAliases(e, next); err != nil {
					return err
				}
			}
		}
	case []any:
		next := p.Next(tree.PathMatchList)
		if aliasParents.Matches(next) || aliasParents.MayContain(next) {
			for _, e := range v {
				if err := promoteAliases(e, next); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// resolveAliasPath returns path with each extension declared in
// extensionAliases replaced by the attribute it stands for. List indexes in
// path match the tree.PathMatchList component of alias parents.
func resolveAliasPath(path tree.Path) tree.Path {
	parts := path.Parts()
	for _, alias := range extensionAliases {
		depth := len(alias.parent.Parts())
		if len(parts) <= depth || parts[depth] != alias.from {
			continue
		}
		if tree.NewPath(withListItems(parts[:depth])...).Matches(alias.parent) {
			parts[depth] = alias.to
		}
	}
	return tree.NewPath(parts...)
}

func withListItems(parts []string) []string {
	normalized := make([]string, len(parts))
	for i, part := range parts {
		normalized[i] = part
		if _, err := strconv.Atoi(part); err == nil {
			normalized[i] = tree.PathMatchList
		}
	}
	return normalized
}
