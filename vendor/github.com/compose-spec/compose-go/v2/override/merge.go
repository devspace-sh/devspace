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

package override

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"

	"github.com/compose-spec/compose-go/v2/tree"
)

// Merge applies overrides to a config model
func Merge(right, left map[string]any) (map[string]any, error) {
	merged, err := MergeYaml(right, left, tree.NewPath())
	if err != nil {
		return nil, err
	}
	return merged.(map[string]any), nil
}

type merger func(any, any, tree.Path) (any, error)

// mergeSpecials defines the custom rules applied by compose when merging yaml trees
var mergeSpecials = map[tree.Path]merger{}

func init() {
	mergeSpecials["networks.*.ipam.config"] = mergeIPAMConfig
	mergeSpecials["networks.*.labels"] = mergeKeyValueSequence
	mergeSpecials["volumes.*.labels"] = mergeKeyValueSequence
	mergeSpecials["secrets.*.labels"] = mergeKeyValueSequence
	mergeSpecials["configs.*.labels"] = mergeKeyValueSequence
	for _, prefix := range []tree.Path{"services", "jobs"} {
		mergeSpecials[prefix+".*.annotations"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.build"] = mergeBuild
		mergeSpecials[prefix+".*.build.args"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.build.additional_contexts"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.build.extra_hosts"] = mergeToSequence
		mergeSpecials[prefix+".*.build.labels"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.command"] = override
		mergeSpecials[prefix+".*.depends_on"] = mergeDependsOn
		mergeSpecials[prefix+".*.dns"] = mergeToSequence
		mergeSpecials[prefix+".*.dns_opt"] = mergeToSequence
		mergeSpecials[prefix+".*.dns_search"] = mergeToSequence
		mergeSpecials[prefix+".*.entrypoint"] = override
		mergeSpecials[prefix+".*.env_file"] = mergeToSequence
		mergeSpecials[prefix+".*.label_file"] = mergeToSequence
		mergeSpecials[prefix+".*.environment"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.extra_hosts"] = mergeToSequence
		mergeSpecials[prefix+".*.healthcheck.test"] = override
		mergeSpecials[prefix+".*.labels"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.volumes.*.volume.labels"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.logging"] = mergeLogging
		mergeSpecials[prefix+".*.models"] = mergeModels
		mergeSpecials[prefix+".*.networks"] = mergeNetworks
		mergeSpecials[prefix+".*.sysctls"] = mergeKeyValueSequence
		mergeSpecials[prefix+".*.tmpfs"] = mergeToSequence
		mergeSpecials[prefix+".*.ulimits.*"] = mergeUlimit
	}
	mergeSpecials["services.*.deploy.labels"] = mergeKeyValueSequence
}

// MergeYaml merges map[string]any yaml trees handling special rules
func MergeYaml(e any, o any, p tree.Path) (any, error) {
	for pattern, merger := range mergeSpecials {
		if p.Matches(pattern) {
			merged, err := merger(e, o, p)
			if err != nil {
				return nil, err
			}
			return merged, nil
		}
	}
	if o == nil {
		return e, nil
	}
	switch value := e.(type) {
	case map[string]any:
		other, ok := o.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot override %s", p)
		}
		return mergeMappings(value, other, p)
	case []any:
		other, ok := o.([]any)
		if !ok {
			return nil, fmt.Errorf("cannot override %s", p)
		}
		return appendWithoutDuplicates(value, other), nil
	default:
		return o, nil
	}
}

func mergeMappings(mapping map[string]any, other map[string]any, p tree.Path) (map[string]any, error) {
	for k, v := range other {
		e, ok := mapping[k]
		if !ok {
			mapping[k] = v
			continue
		}
		next := p.Next(k)
		merged, err := MergeYaml(e, v, next)
		if err != nil {
			return nil, err
		}
		mapping[k] = merged
	}
	return mapping, nil
}

// logging driver options are merged only when both compose file define the same driver
func mergeLogging(config any, other any, p tree.Path) (any, error) {
	base, ok := config.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: cannot merge logging: expected a mapping, got %T", p, config)
	}
	override, ok := other.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: cannot merge logging: expected a mapping, got %T", p, other)
	}
	// we override logging config if source and override have the same driver set, or none
	d, ok1 := override["driver"]
	baseDriver, ok2 := base["driver"]
	if d == baseDriver || !ok1 || !ok2 {
		return mergeMappings(base, override, p)
	}
	return override, nil
}

func mergeBuild(config any, other any, path tree.Path) (any, error) {
	toBuild := func(c any) map[string]any {
		switch v := c.(type) {
		case string:
			return map[string]any{
				"context": v,
			}
		case map[string]any:
			return v
		}
		return nil
	}
	return mergeMappings(toBuild(config), toBuild(other), path)
}

func mergeDependsOn(config any, other any, path tree.Path) (any, error) {
	return mergeAsMapping(config, other, map[string]any{
		"condition": "service_started",
		"required":  true,
	}, path)
}

func mergeModels(config any, other any, path tree.Path) (any, error) {
	return mergeAsMapping(config, other, nil, path)
}

func mergeNetworks(config any, other any, path tree.Path) (any, error) {
	return mergeAsMapping(config, other, nil, path)
}

func mergeAsMapping(config, other any, defaults map[string]any, path tree.Path) (any, error) {
	right, err := convertIntoMapping(nullAsEmpty(config), defaults, path)
	if err != nil {
		return nil, err
	}
	left, err := convertIntoMapping(nullAsEmpty(other), defaults, path)
	if err != nil {
		return nil, err
	}
	return mergeMappings(right, left, path)
}

func mergeToSequence(config any, other any, _ tree.Path) (any, error) {
	right := convertIntoSequence(config)
	left := convertIntoSequence(other)
	return appendWithoutDuplicates(right, left), nil
}

// appendWithoutDuplicates appends override entries to the base sequence,
// ignoring entries strictly identical (deep equality) to one already
// present. Two identical entries never carry more meaning than one, while
// they routinely break things — the same env_file applied twice, a
// duplicate mount rejected by the engine — and dropping them makes merging
// a value over an already-merged result idempotent. Entries that differ in
// form (short vs long syntax of the same thing) are not equal and are kept:
// the rule is strict identity, not equivalence.
func appendWithoutDuplicates(base []any, override []any) []any {
	merged := base
	for _, v := range override {
		if !slices.ContainsFunc(merged, func(existing any) bool { return reflect.DeepEqual(existing, v) }) {
			merged = append(merged, v)
		}
	}
	return merged
}

// mergeKeyValueSequence merges KEY=VALUE lists by key: an override entry
// replaces the base entry of the same key in place, new keys are appended. The
// last entry for a key takes effect, so an override repeating a base entry
// after another value for the same key (`[MODE=debug, MODE=release]` over
// `[MODE=release]`) must end on the repeated one, not drop it as a duplicate.
// Keys repeated in the base collapse the same way, so that the override never
// leaves two identical entries behind.
func mergeKeyValueSequence(config any, other any, path tree.Path) (any, error) {
	return uniqueEntries(slices.Concat(convertIntoSequence(config), convertIntoSequence(other)), keyValueIndexer, path)
}

func convertIntoSequence(value any) []any {
	switch v := value.(type) {
	case map[string]any:
		var seq []any
		for k, val := range v {
			if val == nil {
				seq = append(seq, k)
			} else {
				switch vl := val.(type) {
				// if val is an array we need to add the key with each value one by one
				case []any:
					for _, vlv := range vl {
						seq = append(seq, fmt.Sprintf("%s=%v", k, vlv))
					}
				default:
					seq = append(seq, fmt.Sprintf("%s=%v", k, val))
				}
			}
		}
		slices.SortFunc(seq, func(a, b any) int {
			return cmp.Compare(a.(string), b.(string))
		})
		return seq
	case []any:
		return v
	case string:
		return []any{v}
	}
	return nil
}

func mergeUlimit(config any, other any, path tree.Path) (any, error) {
	over, ismapping := other.(map[string]any)
	if base, ok := config.(map[string]any); ok && ismapping {
		return mergeMappings(base, over, path)
	}
	return other, nil
}

func mergeIPAMConfig(config any, other any, path tree.Path) (any, error) {
	var ipamConfigs []any
	configs, ok := config.([]any)
	if !ok {
		return other, fmt.Errorf("%s: unexpected type %T", path, config)
	}
	overrides, ok := other.([]any)
	if !ok {
		return other, fmt.Errorf("%s: unexpected type %T", path, other)
	}
	for _, original := range configs {
		right, err := convertIntoMapping(original, nil, path)
		if err != nil {
			return nil, err
		}
		for _, override := range overrides {
			left, err := convertIntoMapping(override, nil, path)
			if err != nil {
				return nil, err
			}
			if left["subnet"] != right["subnet"] {
				// check if left is already in ipamConfigs, add it if not and continue with the next config
				if !slices.ContainsFunc(ipamConfigs, func(a any) bool {
					return a.(map[string]any)["subnet"] == left["subnet"]
				}) {
					ipamConfigs = append(ipamConfigs, left)
					continue
				}
			}
			merged, err := mergeMappings(right, left, path)
			if err != nil {
				return nil, err
			}
			// find index of potential previous config with the same subnet in ipamConfigs
			indexIfExist := slices.IndexFunc(ipamConfigs, func(a any) bool {
				return a.(map[string]any)["subnet"] == merged["subnet"]
			})
			// if a previous config is already in ipamConfigs, replace it
			if indexIfExist >= 0 {
				ipamConfigs[indexIfExist] = merged
			} else {
				// or add the new config to ipamConfigs
				ipamConfigs = append(ipamConfigs, merged)
			}
		}
	}
	return ipamConfigs, nil
}

// nullAsEmpty reads a null value as an empty mapping. An empty one rather than
// nil, as merging into a nil map panics.
func nullAsEmpty(value any) any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func convertIntoMapping(a any, defaultValue map[string]any, path tree.Path) (map[string]any, error) {
	switch v := a.(type) {
	case map[string]any:
		return v, nil
	case string:
		if defaultValue == nil {
			return map[string]any{v: nil}, nil
		}
		return map[string]any{v: copyMap(defaultValue)}, nil
	case []any:
		converted := map[string]any{}
		for _, s := range v {
			key, ok := s.(string)
			if !ok {
				return nil, fmt.Errorf("%s: cannot use %T as a mapping key", path, s)
			}
			if defaultValue == nil {
				converted[key] = nil
			} else {
				converted[key] = copyMap(defaultValue)
			}
		}
		return converted, nil
	}
	return nil, fmt.Errorf("%s: cannot convert %T into a mapping", path, a)
}

func copyMap(m map[string]any) map[string]any {
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func override(_ any, other any, _ tree.Path) (any, error) {
	return other, nil
}
