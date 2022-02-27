package loader

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/util"
	dependencyutil "github.com/loft-sh/devspace/pkg/devspace/dependency/util"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/strvals"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

func Imports(basePath string, rawData map[string]interface{}, log log.Logger) (map[string]interface{}, error) {
	rawImports, err := versions.ParseImports(rawData)
	if err != nil {
		return nil, err
	}

	version, ok := rawData["version"].(string)
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
	for idx, i := range imports.Imports {
		configPath, err := dependencyutil.DownloadDependency(basePath, &i.SourceConfig, log)
		if err != nil {
			return nil, errors.Wrap(err, "resolve import")
		}

		fileContent, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, errors.Wrap(err, "read import config")
		}

		rawMap := map[string]interface{}{}
		err = yaml.Unmarshal(fileContent, &rawMap)
		if err != nil {
			return nil, err
		}

		configVersion, ok := rawMap["version"].(string)
		if !ok {
			return nil, fmt.Errorf("version is missing in import config %s", configPath)
		} else if version != configVersion {
			return nil, fmt.Errorf("import mismatch %s != %s. Import at index %d has different version than currently used devspace.yaml, please make sure the versions match between an import and the devspace.yaml using it", version, configVersion, idx)
		}

		delete(rawMap, "version")
		delete(rawMap, "name")
		delete(rawMap, "imports")

		mergedMap = strvals.MergeMapsMergeArrays(mergedMap, rawMap)
	}

	return mergedMap, nil
}
