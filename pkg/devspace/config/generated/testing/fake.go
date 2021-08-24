package testing

import "github.com/loft-sh/devspace/pkg/devspace/config/generated"

// Loader is a fake implementation of the ConigLoader interface
type Loader struct {
	Config generated.Config
}

func (l *Loader) ForDevspace(path string) generated.ConfigLoader {
	return l
}

// Load is a fake implementation o this function
func (l *Loader) Load() (*generated.Config, error) {
	return &l.Config, nil
}

// Save is a fake implementation o this function
func (l *Loader) Save(config *generated.Config) error {
	if config != nil {
		l.Config = *config
	}
	return nil
}
