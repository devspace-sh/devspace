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

package transform

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/tree"
)

// Func is a function that can transform data at a specific path
type Func func(data any, p tree.Path, ignoreParseError bool) (any, error)

var transformers = map[tree.Path]Func{}

func init() {
	// container_spec-level canonicalizations: shared by anything declaring a
	// container — services, jobs, and pre_start init containers
	for _, prefix := range []tree.Path{"services.*", "jobs.*", "services.*.pre_start.*"} {
		transformers[prefix+".env_file"] = transformEnvFile
		transformers[prefix+".label_file"] = transformStringOrList
		transformers[prefix+".gpus"] = transformGpus
		transformers[prefix+".networks"] = transformStringSliceToMap
		transformers[prefix+".models"] = transformStringSliceToMap
		transformers[prefix+".volumes.*"] = transformVolumeMount
		transformers[prefix+".dns"] = transformStringOrList
		transformers[prefix+".devices.*"] = transformDeviceMapping
		transformers[prefix+".secrets.*"] = transformFileMount
		transformers[prefix+".configs.*"] = transformFileMount
		transformers[prefix+".ulimits.*"] = transformUlimits
	}
	// workload_spec and service-level canonicalizations
	for _, prefix := range []tree.Path{"services", "jobs"} {
		transformers[prefix+".*"] = transformService
		transformers[prefix+".*.build.secrets.*"] = transformFileMount
		transformers[prefix+".*.build.provenance"] = transformStringOrX
		transformers[prefix+".*.build.sbom"] = transformStringOrX
		transformers[prefix+".*.build.additional_contexts"] = transformKeyValue
		transformers[prefix+".*.depends_on"] = transformDependsOn
		transformers[prefix+".*.extends"] = transformExtends
		transformers[prefix+".*.ports"] = transformPorts
		transformers[prefix+".*.build"] = transformBuild
		transformers[prefix+".*.build.ssh"] = transformSSH
		transformers[prefix+".*.build.ulimits.*"] = transformUlimits
	}
	transformers["jobs.*.triggers.schedule.*"] = transformSchedule
	transformers["services.*.develop.watch.*.ignore"] = transformStringOrList
	transformers["services.*.develop.watch.*.include"] = transformStringOrList
	transformers["volumes.*"] = transformMaybeExternal
	transformers["networks.*"] = transformMaybeExternal
	transformers["secrets.*"] = transformMaybeExternal
	transformers["configs.*"] = transformMaybeExternal
	transformers["include.*"] = transformInclude
}

func transformStringOrList(data any, _ tree.Path, _ bool) (any, error) {
	switch t := data.(type) {
	case string:
		return []any{t}, nil
	default:
		return data, nil
	}
}

// Canonical transforms a compose model into canonical syntax
func Canonical(yaml map[string]any, ignoreParseError bool) (map[string]any, error) {
	canonical, err := transform(yaml, tree.NewPath(), ignoreParseError)
	if err != nil {
		return nil, err
	}
	return canonical.(map[string]any), nil
}

func transform(data any, p tree.Path, ignoreParseError bool) (any, error) {
	for pattern, transformer := range transformers {
		if p.Matches(pattern) {
			t, err := transformer(data, p, ignoreParseError)
			if err != nil {
				return nil, err
			}
			return t, nil
		}
	}
	switch v := data.(type) {
	case map[string]any:
		a, err := transformMapping(v, p, ignoreParseError)
		if err != nil {
			return a, err
		}
		return v, nil
	case []any:
		a, err := transformSequence(v, p, ignoreParseError)
		if err != nil {
			return a, err
		}
		return v, nil
	default:
		return data, nil
	}
}

func transformSequence(v []any, p tree.Path, ignoreParseError bool) ([]any, error) {
	for i, e := range v {
		t, err := transform(e, p.Next("[]"), ignoreParseError)
		if err != nil {
			return nil, err
		}
		v[i] = t
	}
	return v, nil
}

func transformMapping(v map[string]any, p tree.Path, ignoreParseError bool) (map[string]any, error) {
	for k, e := range v {
		t, err := transform(e, p.Next(k), ignoreParseError)
		if err != nil {
			return nil, err
		}
		v[k] = t
	}
	return v, nil
}

func transformStringOrX(data any, _ tree.Path, _ bool) (any, error) {
	switch v := data.(type) {
	case string:
		return v, nil
	default:
		return fmt.Sprint(v), nil
	}
}
