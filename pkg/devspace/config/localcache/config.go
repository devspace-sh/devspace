package localcache

import (
	"encoding/base64"
	"fmt"
	"github.com/loft-sh/devspace/pkg/util/yamlutil"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/util/encryption"
)

const (
	DevSpaceDisableVarsEncryptionEnv = "DEVSPACE_DISABLE_VARS_ENCRYPTION"
)

// EncryptionKey is the key to encrypt generated variables with. This will be compiled into the binary during the pipeline.
// If empty DevSpace will not encrypt / decrypt the variables.
var EncryptionKey string

// Loader is the interface for loading the cache
type Loader interface {
	Load(devSpaceFilePath string) (Cache, error)
}

type cacheLoader struct{}

// New generates a new generated config
func New(cachePath string) Cache {
	return &LocalCache{
		Vars:   make(map[string]string),
		Images: make(map[string]ImageCache),
		Data:   make(map[string]string),

		cachePath: cachePath,
	}
}

// NewCacheLoader creates a new generated config loader
func NewCacheLoader() Loader {
	return &cacheLoader{}
}

// Load loads the config from the filesystem
func (l *cacheLoader) Load(devSpaceFilePath string) (Cache, error) {
	var loadedConfig *LocalCache

	absPath, err := filepath.Abs(devSpaceFilePath)
	if err != nil {
		return nil, err
	}

	cachePath := cachePath(absPath)
	data, readErr := os.ReadFile(cachePath)
	if readErr != nil {
		loadedConfig = New(cachePath).(*LocalCache)
	} else {
		loadedConfig = &LocalCache{}
		err := yamlutil.Unmarshal(data, loadedConfig)
		if err != nil {
			return nil, err
		}

		if loadedConfig.Images == nil {
			loadedConfig.Images = make(map[string]ImageCache)
		}
		if loadedConfig.Data == nil {
			loadedConfig.Data = make(map[string]string)
		}
		if loadedConfig.Vars == nil {
			loadedConfig.Vars = make(map[string]string)
		}
	}

	// Decrypt vars if necessary
	if loadedConfig.VarsEncrypted {
		for k, v := range loadedConfig.Vars {
			if len(v) == 0 {
				continue
			}

			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				// seems like not encrypted
				continue
			}

			decrypted, err := encryption.DecryptAES([]byte(EncryptionKey), decoded)
			if err != nil {
				// we cannot decrypt the variable, so we will ask the user again
				delete(loadedConfig.Vars, k)
				continue
			}

			loadedConfig.Vars[k] = string(decrypted)
		}

		loadedConfig.VarsEncrypted = false
	}

	loadedConfig.cachePath = cachePath
	return loadedConfig, nil
}

// cachePath returns the cache absolute path. The if the default devspace.yaml is given the cache path
// will be $PWD/.devspace/cache.yaml. For any other file name it will be $PWD/.devspace/cache-[file name]
func cachePath(devSpaceConfigPath string) string {
	fileDir := filepath.Dir(devSpaceConfigPath)
	fileName := filepath.Base(devSpaceConfigPath)
	if fileName == constants.DefaultConfigPath {
		return filepath.Join(fileDir, constants.DefaultCacheFolder, "cache.yaml")
	}

	return filepath.Join(fileDir, constants.DefaultCacheFolder, fmt.Sprintf("cache-%s", fileName))
}
