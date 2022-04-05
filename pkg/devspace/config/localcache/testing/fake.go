package testing

import "github.com/loft-sh/devspace/pkg/devspace/config/localcache"

// Loader is a fake implementation of the ConigLoader interface
type Loader struct {
	Config *localcache.LocalCache
}

func (l *Loader) ForDevspace(path string) localcache.Loader {
	return l
}

// Load is a fake implementation o this function
func (l *Loader) Load(devSpaceFilePath string) (localcache.Cache, error) {
	return l.Config, nil
}

// Save is a fake implementation o this function
func (l *Loader) Save(config *localcache.LocalCache) error {
	if config != nil {
		l.Config = config
	}
	return nil
}
