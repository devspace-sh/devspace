package localcache

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/loft-sh/devspace/pkg/devspace/env"
	"github.com/loft-sh/devspace/pkg/util/encryption"
	"gopkg.in/yaml.v3"
)

type Cache interface {
	ListImageCache() map[string]ImageCache
	GetImageCache(imageConfigName string) (ImageCache, bool)
	SetImageCache(imageConfigName string, imageCache ImageCache)

	GetLastContext() *LastContextConfig
	SetLastContext(config *LastContextConfig)

	GetData(key string) (string, bool)
	SetData(key, value string)

	GetVar(varName string) (string, bool)
	SetVar(varName, value string)
	ListVars() map[string]string
	ClearVars()

	DeepCopy() Cache

	// Save persists changes to file
	Save() error
}

// LocalCache specifies the runtime cache
type LocalCache struct {
	Vars          map[string]string `yaml:"vars,omitempty"`
	VarsEncrypted bool              `yaml:"varsEncrypted,omitempty"`

	Images      map[string]ImageCache `yaml:"images,omitempty"`
	LastContext *LastContextConfig    `yaml:"lastContext,omitempty"`

	// Data is arbitrary key value cache
	Data map[string]string `yaml:"data,omitempty"`

	// config path is the path where the cache was loaded from
	cachePath   string     `yaml:"-" json:"-"`
	accessMutex sync.Mutex `yaml:"-" json:"-"`
}

// LastContextConfig holds all the informations about the last used kubernetes context
type LastContextConfig struct {
	Namespace string `yaml:"namespace,omitempty"`
	Context   string `yaml:"context,omitempty"`
}

// ImageCache holds the cache related information about a certain image
type ImageCache struct {
	ImageConfigHash string `yaml:"imageConfigHash,omitempty"`

	DockerfileHash string `yaml:"dockerfileHash,omitempty"`
	ContextHash    string `yaml:"contextHash,omitempty"`
	EntrypointHash string `yaml:"entrypointHash,omitempty"`

	CustomFilesHash string `yaml:"customFilesHash,omitempty"`

	ImageName              string `yaml:"imageName,omitempty"`
	LocalRegistryImageName string `yaml:"localRegistryImageName,omitempty"`
	Tag                    string `yaml:"tag,omitempty"`
}

func (ic ImageCache) IsLocalRegistryImage() bool {
	return ic.LocalRegistryImageName != ""
}

func (ic ImageCache) ResolveImage() string {
	if ic.IsLocalRegistryImage() {
		return ic.LocalRegistryImageName
	}

	return ic.ImageName
}

func (l *LocalCache) ListImageCache() map[string]ImageCache {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	retMap := map[string]ImageCache{}
	for k, v := range l.Images {
		retMap[k] = v
	}

	return retMap
}

func (l *LocalCache) GetImageCache(imageConfigName string) (ImageCache, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	cache, ok := l.Images[imageConfigName]
	return cache, ok
}

func (l *LocalCache) SetImageCache(imageConfigName string, imageCache ImageCache) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.Images[imageConfigName] = imageCache
}

func (l *LocalCache) GetLastContext() *LastContextConfig {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	return l.LastContext
}

func (l *LocalCache) SetLastContext(config *LastContextConfig) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.LastContext = config
}

func (l *LocalCache) GetData(key string) (string, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	cache, ok := l.Data[key]
	return cache, ok
}

func (l *LocalCache) SetData(key, value string) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.Data[key] = value
}

func (l *LocalCache) GetVar(varName string) (string, bool) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	cache, ok := l.Vars[varName]
	return cache, ok
}

func (l *LocalCache) SetVar(varName, value string) {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.Vars[varName] = value
}

func (l *LocalCache) ListVars() map[string]string {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	listVars := map[string]string{}
	for k, v := range l.Vars {
		listVars[k] = v
	}
	return listVars
}

func (l *LocalCache) ClearVars() {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	l.Vars = map[string]string{}
}

// DeepCopy creates a deep copy of the config
func (l *LocalCache) DeepCopy() Cache {
	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	o, _ := yaml.Marshal(l)
	n := &LocalCache{}
	_ = yaml.Unmarshal(o, n)
	n.cachePath = l.cachePath
	return n
}

// Save saves the config to the filesystem
func (l *LocalCache) Save() error {
	if l.cachePath == "" {
		return fmt.Errorf("no path specified where to save the local cache")
	}

	l.accessMutex.Lock()
	defer l.accessMutex.Unlock()

	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}

	copiedConfig := &LocalCache{}
	err = yaml.Unmarshal(data, copiedConfig)
	if err != nil {
		return err
	}

	// encrypt variables
	if env.GlobalGetEnv(DevSpaceDisableVarsEncryptionEnv) != "true" && EncryptionKey != "" {
		for k, v := range copiedConfig.Vars {
			if len(v) == 0 {
				continue
			}

			encrypted, err := encryption.EncryptAES([]byte(EncryptionKey), []byte(v))
			if err != nil {
				return err
			}

			copiedConfig.Vars[k] = base64.StdEncoding.EncodeToString(encrypted)
		}

		copiedConfig.VarsEncrypted = true
	}

	// marshal again with the encrypted vars
	data, err = yaml.Marshal(copiedConfig)
	if err != nil {
		return err
	}

	_, err = os.Stat(l.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// check if a save is really necessary
			if len(l.Data) == 0 && len(l.Vars) == 0 && len(l.Images) == 0 && l.LastContext == nil {
				return nil
			}
		}
	}

	err = os.MkdirAll(filepath.Dir(l.cachePath), 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(l.cachePath, data, 0666)
}
