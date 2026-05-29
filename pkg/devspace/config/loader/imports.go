package loader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
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

const maxConcurrentImportDownloads = 8

type resolvedImport struct {
	ConfigPath string
	Data       map[string]interface{}
	Disabled   bool
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

	resolvedImports, err := loadImports(ctx, basePath, imports.Imports, version, log)
	if err != nil {
		return nil, err
	}

	mergedMap := map[string]interface{}{}
	err = util.Convert(rawData, mergedMap)
	if err != nil {
		return nil, err
	}

	// load imports
	for _, resolvedImport := range resolvedImports {
		if resolvedImport.Disabled {
			continue
		}

		importData := resolvedImport.Data

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

			// The recursive call starts by reloading variables from mergedMap.
			mergedMap, err = ResolveImports(ctx, resolver, filepath.Dir(resolvedImport.ConfigPath), mergedMap, log)
			if err != nil {
				return nil, err
			}
		} else {
			delete(mergedMap, "imports")

			// Leaf imports used to recurse too, so preserve the variable reload
			// after each ordered import merge.
			err = reloadVariables(resolver, mergedMap, log)
			if err != nil {
				return nil, err
			}
		}
	}

	return mergedMap, nil
}

func loadImports(ctx context.Context, basePath string, imports []latest.Import, version string, log log.Logger) ([]resolvedImport, error) {
	resolvedImports := make([]resolvedImport, len(imports))

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrentImportDownloads)
	for index := range imports {
		// Keep an explicit per-iteration copy for the goroutine closure.
		importConfig := imports[index]
		if importConfig.Enabled != nil && !*importConfig.Enabled {
			resolvedImports[index] = resolvedImport{Disabled: true}
			continue
		}

		eg.Go(func() error {
			configPath, err := dependencyutil.DownloadDependency(ctx, basePath, &importConfig.SourceConfig, log)
			if err != nil {
				return errors.Wrap(err, "resolve import")
			}

			fileContent, err := os.ReadFile(configPath)
			if err != nil {
				return errors.Wrap(err, "read import config")
			}

			importData := map[string]interface{}{}
			err = yamlutil.Unmarshal(fileContent, &importData)
			if err != nil {
				return err
			}

			configVersion, ok := importData["version"].(string)
			if !ok {
				return fmt.Errorf("version is missing in import config %s", configPath)
			} else if version != configVersion {
				return fmt.Errorf("import mismatch %s != %s. Import %s has different version than currently used devspace.yaml, please make sure the versions match between an import and the devspace.yaml using it", version, configVersion, configPath)
			}

			resolvedImports[index] = resolvedImport{
				ConfigPath: configPath,
				Data:       importData,
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return resolvedImports, nil
}
