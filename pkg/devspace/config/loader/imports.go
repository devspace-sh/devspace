package loader

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"path/filepath"
)

var ImportSections = []string{
	"vars",
	"dev",
	"deployments",
	"images",
	"pipelines",
	"commands",
	"pullSecrets",
	"dependencies",
}

func ResolveImports(ctx context.Context, basePath string, rawData map[string]interface{}, log log.Logger) (map[string]interface{}, error) {
	rawImports, err := versions.GetImports(rawData)
	if err != nil {
		return nil, err
	}

	version, ok := rawImports["version"].(string)
	if !ok {
		return nil, errors.Errorf("Version is missing in devspace.yaml")
	}

	imports, err := versions.Parse(rawImports, log)
	if err != nil {
		return nil, err
	}

	err = Validate(imports)
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
		configPath, err := dependencyutil.DownloadDependency(ctx, basePath, &i.SourceConfig, log)
		if err != nil {
			return nil, errors.Wrap(err, "resolve import")
		}

		fileContent, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, errors.Wrap(err, "read import config")
		}

		importData := map[string]interface{}{}
		err = yaml.Unmarshal(fileContent, &importData)
		if err != nil {
			return nil, err
		}

		configVersion, ok := importData["version"].(string)
		if !ok {
			return nil, fmt.Errorf("version is missing in import config %s", configPath)
		} else if version != configVersion {
			return nil, fmt.Errorf("import mismatch %s != %s. Import %s has different version than currently used devspace.yaml, please make sure the versions match between an import and the devspace.yaml using it", version, configVersion, configPath)
		}

		// resolve the import imports
		resolvedMap, err := ResolveImports(ctx, filepath.Dir(configPath), importData, log)
		if err != nil {
			return nil, err
		}

		// merge sections
		for _, section := range ImportSections {
			sectionMap, ok := resolvedMap[section].(map[string]interface{})
			if !ok {
				continue
			}

			// make sure the section exists
			if mergedMap[section] == nil {
				mergedMap[section] = map[string]interface{}{}
			}

			for key, value := range sectionMap {
				_, ok := mergedMap[section].(map[string]interface{})[key]
				if ok {
					return nil, fmt.Errorf("cannot import %s: section %s already has an item with key %s. Please make sure that imported %s keys do not collide across the current config and imported configs", configPath, section, key, section)
				}

				mergedMap[section].(map[string]interface{})[key] = value
			}
		}
	}

	return mergedMap, nil
}
