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
	"context"
	"crypto/sha256"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/v2/dotenv"
	interp "github.com/compose-spec/compose-go/v2/interpolation"
	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/tree"
	"github.com/compose-spec/compose-go/v2/types"
)

type includeCacheKey struct{}

// withIncludeCache attaches a fresh include cache to ctx. The cache is scoped
// to a single load so a file reachable through several include paths (a
// "diamond" include graph) is parsed and expanded only once per distinct
// (paths, directories, environment) tuple.
func withIncludeCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, includeCacheKey{}, map[string]map[string]any{})
}

func includeCache(ctx context.Context) map[string]map[string]any {
	cache, _ := ctx.Value(includeCacheKey{}).(map[string]map[string]any)
	return cache
}

// includeModelKey covers every input that determines the model loaded for an
// include: resolved paths, working directory, project directory and effective
// environment. Interpolate.Substitute and TypeCastMapping are intentionally
// excluded: they are invariant across includes within a single load. If a
// future option allows them to vary per include, they must be folded into the
// key. Fields are length-prefixed and collections count-prefixed so the byte
// stream is uniquely decodable.
func includeModelKey(paths types.StringList, workingDir, projectDir string, env types.Mapping) string {
	h := sha256.New()
	write := func(s string) {
		fmt.Fprintf(h, "%d:%s", len(s), s)
	}
	fmt.Fprintf(h, "%d;", len(paths))
	for _, p := range paths {
		write(p)
	}
	write(workingDir)
	write(projectDir)
	fmt.Fprintf(h, "%d;", len(env))
	for _, k := range slices.Sorted(maps.Keys(env)) {
		write(k)
		write(env[k])
	}
	return string(h.Sum(nil))
}

// loadIncludeConfig parse the required config from raw yaml
func loadIncludeConfig(source any) ([]types.IncludeConfig, error) {
	if source == nil {
		return nil, nil
	}
	configs, ok := source.([]any)
	if !ok {
		return nil, fmt.Errorf("`include` must be a list, got %s", source)
	}
	for i, config := range configs {
		if v, ok := config.(string); ok {
			configs[i] = map[string]any{
				"path": v,
			}
		}
	}
	var requires []types.IncludeConfig
	err := Transform(source, &requires)
	return requires, err
}

func ApplyInclude(ctx context.Context, workingDir string, environment types.Mapping, model map[string]any, options *Options, included []string, processor PostProcessor) error {
	includeConfig, err := loadIncludeConfig(model["include"])
	if err != nil {
		return err
	}

	for _, r := range includeConfig {
		for _, listener := range options.Listeners {
			listener("include", map[string]any{
				"path":       r.Path,
				"workingdir": workingDir,
			})
		}

		var relworkingdir string
		for i, p := range r.Path {
			for _, loader := range options.ResourceLoaders {
				if !loader.Accept(p) {
					continue
				}
				path, err := loader.Load(ctx, p)
				if err != nil {
					return err
				}
				p = path

				if i == 0 { // This is the "main" file, used to define project-directory. Others are overrides

					switch {
					case r.ProjectDirectory == "":
						relworkingdir = loader.Dir(path)
						r.ProjectDirectory = filepath.Dir(path)
					case !filepath.IsAbs(r.ProjectDirectory):
						relworkingdir = loader.Dir(r.ProjectDirectory)
						r.ProjectDirectory = filepath.Join(workingDir, r.ProjectDirectory)

					default:
						relworkingdir = r.ProjectDirectory

					}
					for _, f := range included {
						if f == path {
							included = append(included, path)
							return fmt.Errorf("include cycle detected:\n%s\n include %s", included[0], strings.Join(included[1:], "\n include "))
						}
					}
				}
			}
			r.Path[i] = p
		}

		loadOptions := options.clone()
		loadOptions.ResolvePaths = true
		loadOptions.SkipNormalization = true
		loadOptions.SkipConsistencyCheck = true
		// include and extends events are only reported for declarations in the
		// config files passed to the loader, not for those in included files
		loadOptions.Listeners = nil
		loadOptions.ResourceLoaders = append(loadOptions.RemoteResourceLoaders(), localResourceLoader{
			WorkingDir: r.ProjectDirectory,
		})

		if len(r.EnvFile) == 0 {
			f := filepath.Join(r.ProjectDirectory, ".env")
			if s, err := os.Stat(f); err == nil && !s.IsDir() {
				r.EnvFile = types.StringList{f}
			}
		} else {
			envFile := []string{}
			for _, f := range r.EnvFile {
				if f == "/dev/null" {
					continue
				}
				if !filepath.IsAbs(f) {
					f = filepath.Join(workingDir, f)
					s, err := os.Stat(f)
					if err != nil {
						return err
					}
					if s.IsDir() {
						return fmt.Errorf("%s is not a file", f)
					}
				}
				envFile = append(envFile, f)
			}
			r.EnvFile = envFile
		}

		envFromFile, err := dotenv.GetEnvFromFile(environment, r.EnvFile)
		if err != nil {
			return err
		}

		config := types.ConfigDetails{
			WorkingDir:  relworkingdir,
			ConfigFiles: types.ToConfigFiles(r.Path),
			Environment: environment.Clone().Merge(envFromFile),
		}
		loadOptions.Interpolate = &interp.Options{
			Substitute:      options.Interpolate.Substitute,
			LookupValue:     config.LookupEnv,
			TypeCastMapping: options.Interpolate.TypeCastMapping,
		}
		cache := includeCache(ctx)
		cacheKey := includeModelKey(r.Path, relworkingdir, r.ProjectDirectory, config.Environment)
		imported, hit := cache[cacheKey]
		if hit {
			// hand out a copy, the cached model must stay pristine
			imported = deepClone(imported).(map[string]any)
		} else {
			imported, err = loadYamlModel(ctx, config, loadOptions, &cycleTracker{}, included)
			if err != nil {
				return err
			}
			if cache != nil {
				// store a pristine copy: the model returned to the caller is
				// merged into the parent and mutated by later loading phases
				cache[cacheKey] = deepClone(imported).(map[string]any)
			}
		}
		err = importResources(imported, model, processor)
		if err != nil {
			return err
		}
	}
	delete(model, "include")
	return nil
}

// importResources import into model all resources defined by imported, and report error on conflict
func importResources(source map[string]any, target map[string]any, processor PostProcessor) error {
	for _, key := range []string{"services", "jobs", "volumes", "networks", "secrets", "configs", "models"} {
		if err := importResource(source, target, key, processor); err != nil {
			return err
		}
	}
	return nil
}

func importResource(source map[string]any, target map[string]any, key string, processor PostProcessor) error {
	from := source[key]
	if from != nil {
		var to map[string]any
		if v, ok := target[key]; ok {
			to = v.(map[string]any)
		} else {
			to = map[string]any{}
		}
		for name, a := range from.(map[string]any) {
			conflict, ok := to[name]
			if !ok {
				to[name] = a
				continue
			}
			if reflect.DeepEqual(a, conflict) {
				// Same resource reached through multiple include paths (a
				// diamond); re-merging identical definitions would append
				// duplicate entries to list-valued fields.
				continue
			}
			err := processor.Apply(map[string]any{
				key: map[string]any{
					name: a,
				},
			})
			if err != nil {
				return err
			}

			merged, err := override.MergeYaml(a, conflict, tree.NewPath(key, name))
			if err != nil {
				return err
			}
			to[name] = merged
		}
		target[key] = to
	}
	return nil
}
