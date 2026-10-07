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
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/tree"
	"go.yaml.in/yaml/v4"
)

// defaultMaxNodeVisits caps total resolveReset calls per document.
// Sized to accommodate large real-world compose files while rejecting documents that would
// cause unbounded traversal. Callers can override this via Options.MaxNodeVisits.
const defaultMaxNodeVisits = 100_000

// nodeCache stores a resolved node and the relative sub-paths within its subtree that
// carried !reset/!override tags, so cache hits at different call sites can replay them.
type nodeCache struct {
	node          *yaml.Node
	relativePaths []tree.Path
}

type ResetProcessor struct {
	target        any
	paths         []tree.Path
	visitedNodes  map[*yaml.Node][]tree.Path
	resolvedNodes map[*yaml.Node]nodeCache
	visitCount    int
	// maxNodeVisits is the per-document cap; when zero, defaultMaxNodeVisits is used.
	maxNodeVisits int
}

// UnmarshalYAML implement yaml.Unmarshaler
func (p *ResetProcessor) UnmarshalYAML(value *yaml.Node) error {
	p.visitedNodes = make(map[*yaml.Node][]tree.Path)
	p.resolvedNodes = make(map[*yaml.Node]nodeCache)
	p.visitCount = 0
	defer func() {
		p.visitedNodes = nil
		p.resolvedNodes = nil
	}()
	resolved, err := p.resolveReset(value, tree.NewPath())
	if err != nil {
		return err
	}
	p.addAliasPaths()
	return resolved.Decode(p.target)
}

// addAliasPaths records, next to each path going through an extension declared
// in extensionAliases, the path of the attribute it stands for: earlier files
// hold that attribute under its own name once their extensions are promoted.
func (p *ResetProcessor) addAliasPaths() {
	for _, path := range p.paths {
		if resolved := resolveAliasPath(path); resolved != path {
			p.paths = append(p.paths, resolved)
		}
	}
}

// resolveReset detects `!reset` tag being set on yaml nodes and record position in the yaml tree
func (p *ResetProcessor) resolveReset(node *yaml.Node, path tree.Path) (*yaml.Node, error) {
	p.visitCount++
	limit := p.maxNodeVisits
	if limit <= 0 {
		limit = defaultMaxNodeVisits
	}
	if p.visitCount > limit {
		return nil, fmt.Errorf("compose file exceeds maximum node visit limit (%d)", limit)
	}

	pathStr := path.String()
	// If the path contains "<<", removing the "<<" element and merging the path
	if strings.Contains(pathStr, ".<<") {
		path = tree.NewPath(strings.Replace(pathStr, ".<<", "", 1))
	}

	if node.Tag == "!reset" {
		p.paths = append(p.paths, path)
		return nil, nil
	}
	if node.Tag == "!override" {
		p.paths = append(p.paths, path)
		return node, nil
	}

	// If the node is an alias, process the alias target via the cache so each anchor is
	// processed at most once.
	if node.Kind == yaml.AliasNode {
		if err := p.checkForCycle(node.Alias, path); err != nil {
			return nil, err
		}
		// Handle !reset/!override on the alias target before delegating to the cache,
		// keeping all tag-handling logic in resolveReset rather than split across functions.
		target := node.Alias
		if target.Tag == "!reset" {
			p.paths = append(p.paths, path)
			return nil, nil
		}
		if target.Tag == "!override" {
			p.paths = append(p.paths, path)
			return target, nil
		}
		return p.cachedResolve(target, path)
	}

	// Container nodes are resolved through the cache, ensuring resolved containers are
	// not re-traversed.
	if node.Kind == yaml.SequenceNode || node.Kind == yaml.MappingNode {
		return p.cachedResolve(node, path)
	}

	return node, nil
}

// cachedResolve resolves node (a container without !reset/!override), serving from cache on
// repeat visits to prevent re-traversal. It is only called after tag checks are done in
// resolveReset, so it never receives !reset/!override-tagged nodes.
func (p *ResetProcessor) cachedResolve(node *yaml.Node, path tree.Path) (*yaml.Node, error) {
	if cached, ok := p.resolvedNodes[node]; ok {
		for _, rel := range cached.relativePaths {
			p.paths = append(p.paths, joinPath(path, rel))
		}
		return cached.node, nil
	}

	startIdx := len(p.paths)
	resolved, err := p.resolveContainer(node, path)
	if err != nil {
		return nil, err
	}

	var relPaths []tree.Path
	for _, addedPath := range p.paths[startIdx:] {
		rel, err := subPath(addedPath, path)
		if err != nil {
			return nil, err
		}
		relPaths = append(relPaths, rel)
	}
	p.resolvedNodes[node] = nodeCache{node: resolved, relativePaths: relPaths}
	return resolved, nil
}

// resolveContainer processes the children of a Sequence or Mapping node.
// AliasNodes must be kept as-is in the output Content; the resolved value is used only
// for tag inspection. Changing this will affect how the YAML library handles the document
// during decoding.
func (p *ResetProcessor) resolveContainer(node *yaml.Node, path tree.Path) (*yaml.Node, error) {
	switch node.Kind {
	case yaml.SequenceNode:
		var nodes []*yaml.Node
		for idx, v := range node.Content {
			next := path.Next(strconv.Itoa(idx))
			resolved, err := p.resolveReset(v, next)
			if err != nil {
				return nil, err
			}
			if resolved == nil {
				continue
			}
			if v.Kind == yaml.AliasNode {
				nodes = append(nodes, v)
			} else {
				nodes = append(nodes, resolved)
			}
		}
		node.Content = nodes
	case yaml.MappingNode:
		keys := map[string]int{}
		var key string
		var nodes []*yaml.Node
		for idx, v := range node.Content {
			if idx%2 == 0 {
				key = v.Value
				if line, seen := keys[key]; seen {
					return nil, fmt.Errorf("line %d: mapping key %#v already defined at line %d", v.Line, key, line)
				}
				keys[key] = v.Line
			} else {
				resolved, err := p.resolveReset(v, path.Next(key))
				if err != nil {
					return nil, err
				}
				if resolved == nil {
					continue
				}
				// Under the merge key `<<`, the YAML library only accepts an
				// AliasNode value when its target is a MappingNode. An alias to a
				// SequenceNode (the spec-allowed "sequence of mappings" form via an
				// anchor) is rejected. Substitute the resolved target so the YAML
				// library sees the underlying node directly for merge keys.
				if v.Kind == yaml.AliasNode && key != "<<" {
					nodes = append(nodes, node.Content[idx-1], v)
				} else {
					nodes = append(nodes, node.Content[idx-1], resolved)
				}
			}
		}
		node.Content = nodes
	}
	return node, nil
}

// subPath strips base from full to produce a relative path for cache storage.
// Returns "" when full == base (the !reset/!override tag is on the node root itself).
// Returns an error when full is not rooted at base, which would indicate a logic error
// in resolveReset/cachedResolve.
func subPath(full, base tree.Path) (tree.Path, error) {
	if base == "" {
		return full, nil
	}
	fullStr := string(full)
	baseStr := string(base)
	if fullStr == baseStr {
		return "", nil
	}
	prefix := baseStr + "."
	if strings.HasPrefix(fullStr, prefix) {
		return tree.Path(fullStr[len(prefix):]), nil
	}
	return "", fmt.Errorf("internal error: path %q is not a sub-path of %q", fullStr, baseStr)
}

// joinPath reconstructs an absolute path from a call-site base and a cached relative path.
// A relative path of "" means the tag was on the node root, so base is returned unchanged.
func joinPath(base, rel tree.Path) tree.Path {
	if rel == "" {
		return base
	}
	if base == "" {
		return rel
	}
	return tree.Path(string(base) + "." + string(rel))
}

// Apply finds the go attributes matching recorded paths and reset them to zero value
func (p *ResetProcessor) Apply(target any) error {
	return p.applyNullOverrides(target, tree.NewPath())
}

// applyNullOverrides set val to Zero if it matches any of the recorded paths
func (p *ResetProcessor) applyNullOverrides(target any, path tree.Path) error {
	switch v := target.(type) {
	case map[string]any:
	KEYS:
		for k, e := range v {
			next := path.Next(k)
			for _, pattern := range p.paths {
				if next.Matches(pattern) {
					delete(v, k)
					continue KEYS
				}
			}
			err := p.applyNullOverrides(e, next)
			if err != nil {
				return err
			}
		}
	case []any:
	ITER:
		for i, e := range v {
			next := path.Next(fmt.Sprintf("[%d]", i))
			for _, pattern := range p.paths {
				if next.Matches(pattern) {
					continue ITER
					// TODO(ndeloof) support removal from sequence
				}
			}
			err := p.applyNullOverrides(e, next)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *ResetProcessor) checkForCycle(node *yaml.Node, path tree.Path) error {
	paths := p.visitedNodes[node]

	for _, prevPath := range paths {
		// If we're visiting the exact same path, it's not a cycle
		if path == prevPath {
			continue
		}

		// Compare on the raw form so dots inside escaped segment names (e.g.
		// service names containing ".") aren't conflated with path separators.
		pathStr := string(path)
		prevStr := string(prevPath)

		// If either path is using a merge key, it's legitimate YAML merging
		if strings.Contains(prevStr, "<<") || strings.Contains(pathStr, "<<") {
			continue
		}

		// Only consider it a cycle if one path is contained within the other
		// and they're not in different service definitions
		if (strings.HasPrefix(pathStr, prevStr+".") ||
			strings.HasPrefix(prevStr, pathStr+".")) &&
			!areInDifferentServices(path, prevPath) {
			return fmt.Errorf("cycle detected: node at path %s references node at path %s",
				path.String(), prevPath.String())
		}
	}

	p.visitedNodes[node] = append(paths, path)
	return nil
}

// areInDifferentServices checks if two paths are in different service definitions
func areInDifferentServices(path1, path2 tree.Path) bool {
	parts1 := path1.Parts()
	parts2 := path2.Parts()
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] == "services" && i+1 < len(parts1) &&
			parts2[i] == "services" && i+1 < len(parts2) {
			return parts1[i+1] != parts2[i+1]
		}
	}
	return false
}
