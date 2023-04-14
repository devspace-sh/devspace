package loader

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
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

			switch section {

			// special handling of require section to get required commands appended from all of the imports
			case "require":
				const (
					devspaceConstraintKey = "devspace"
					commandConstraintsKey = "commands"
					pluginConstraintsKey  = "plugins"
				)

				// check devspace version constraints
				currDevspaceVersionConst, currHasConstr :=
					mergedMap[section].(map[string]interface{})[devspaceConstraintKey].(string)
				if currDevspaceVersionConst == "" {
					currHasConstr = false
				}

				// check import devspace version constraint
				importDevspaceVersionConst, importHasConstr := sectionMap[devspaceConstraintKey].(string)
				if importDevspaceVersionConst == "" {
					importHasConstr = false
				}

				// set the constraint from import if the current is empty
				if !currHasConstr && importHasConstr {
					mergedMap[section].(map[string]interface{})[devspaceConstraintKey] = importDevspaceVersionConst
				}

				// append the constraint if it's not already present in the string
				if currHasConstr &&
					importHasConstr &&
					!strings.Contains(currDevspaceVersionConst, importDevspaceVersionConst) {
					mergedMap[section].(map[string]interface{})[devspaceConstraintKey] =
						fmt.Sprintf("%s, %s", currDevspaceVersionConst, importDevspaceVersionConst)
				}

				// handle command constraints by appending them to the current set of constraints
				importCommandConstraints, ok := sectionMap[commandConstraintsKey].([]interface{})
				if ok {

					_, ok := mergedMap[section].(map[string]interface{})[commandConstraintsKey].([]interface{})
					if !ok {
						mergedMap[section].(map[string]interface{})[commandConstraintsKey] = []interface{}{}
					}

					mergedMap[section].(map[string]interface{})[commandConstraintsKey] = append(
						mergedMap[section].(map[string]interface{})[commandConstraintsKey].([]interface{}),
						importCommandConstraints...,
					)
				}

				// handle plugin constraints by appending them to the current set of constraints
				importPluginConstraints, ok := sectionMap[pluginConstraintsKey].([]interface{})
				if ok {

					_, ok := mergedMap[section].(map[string]interface{})[pluginConstraintsKey].([]interface{})
					if !ok {
						mergedMap[section].(map[string]interface{})[pluginConstraintsKey] = []interface{}{}
					}

					mergedMap[section].(map[string]interface{})[pluginConstraintsKey] = append(
						mergedMap[section].(map[string]interface{})[pluginConstraintsKey].([]interface{}),
						importPluginConstraints...,
					)
				}

			default:
				for key, value := range sectionMap {
					_, ok := mergedMap[section].(map[string]interface{})[key]
					if !ok {
						mergedMap[section].(map[string]interface{})[key] = value
					}
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
