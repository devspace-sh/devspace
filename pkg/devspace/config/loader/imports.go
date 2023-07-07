package loader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
)

var ImportSections = []string{
	"require",
	"vars",
	"dev",
	"deployments",
	"images",
	"pipelines",
	"commands",
	"functions",
	"pullSecrets",
	"dependencies",
	"profiles",
	"hooks",
	"localRegistry",
}

func ResolveImports(ctx context.Context, resolver variable.Resolver, basePath string, rawData map[string]interface{}, log log.Logger) (map[string]interface{}, error) {
	// initially reload variables
	err := reloadVariables(resolver, rawData, log)
	if err != nil {
		return nil, err
	}

	rawImports, err := versions.Get(rawData, "imports")
	if err != nil {
		return nil, err
	}

	version, ok := rawImports["version"].(string)
	if !ok {
		return nil, errors.Errorf("Version is missing in devspace.yaml")
	}

	rawImportsInterface, err := resolver.FillVariablesInclude(ctx, rawImports, true, []string{"/imports/**"})
	if err != nil {
		return nil, err
	}

	rawImports = rawImportsInterface.(map[string]interface{})
	imports, err := versions.Parse(rawImports, log)
	if err != nil {
		return nil, err
	}

	mergedMap := map[string]interface{}{}
	err = util.Convert(rawData, mergedMap)
	if err != nil {
		return nil, err
	}

	// load imports
	for _, i := range imports.Imports {
		if i.Enabled != nil && !*i.Enabled {
			continue
		}

		configPath, err := dependencyutil.DownloadDependency(ctx, basePath, &i.SourceConfig, log)
		if err != nil {
			return nil, errors.Wrap(err, "resolve import")
		}

		fileContent, err := os.ReadFile(configPath)
		if err != nil {
			return nil, errors.Wrap(err, "read import config")
		}

		importData := map[string]interface{}{}
		err = yamlutil.Unmarshal(fileContent, &importData)
		if err != nil {
			return nil, err
		}

		configVersion, ok := importData["version"].(string)
		if !ok {
			return nil, fmt.Errorf("version is missing in import config %s", configPath)
		} else if version != configVersion {
			return nil, fmt.Errorf("import mismatch %s != %s. Import %s has different version than currently used devspace.yaml, please make sure the versions match between an import and the devspace.yaml using it", version, configVersion, configPath)
		}

		// merge sections
		for _, section := range ImportSections {
			sectionMap, ok := importData[section].(map[string]interface{})
			if !ok {
				// no map, is it a slice?
				sectionSlice, ok := importData[section].([]interface{})
				if !ok {
					continue
				}

				// make sure the section exists
				if mergedMap[section] == nil {
					mergedMap[section] = []interface{}{}
				}
				for _, value := range sectionSlice {
					mergedMap[section] = append(mergedMap[section].([]interface{}), value)
				}
				continue
			}

			// make sure the section exists
			if mergedMap[section] == nil {
				mergedMap[section] = map[string]interface{}{}
			}

			for key, value := range sectionMap {
				_, ok := mergedMap[section].(map[string]interface{})[key]
				if !ok {
					mergedMap[section].(map[string]interface{})[key] = value
				}
			}
		}

		// resolve the import imports
		if importData["imports"] != nil {
			mergedMap["imports"] = importData["imports"]
		} else {
			delete(mergedMap, "imports")
		}

		// resolve imports
		mergedMap, err = ResolveImports(ctx, resolver, filepath.Dir(configPath), mergedMap, log)
		if err != nil {
			return nil, err
		}
	}

	return mergedMap, nil
}
