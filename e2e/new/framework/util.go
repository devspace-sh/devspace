package framework

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"sync"
)

func LoadConfig(f factory.Factory, configPath string) (config.Config, []types.Dependency, error) {
	before, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	defer SwitchDir(before)

	// Set config root
	log := f.GetLog()
	configOptions := &loader.ConfigOptions{}
	configLoader := f.NewConfigLoader(configPath)
	configExists, err := configLoader.SetDevSpaceRoot(log)
	if err != nil {
		return nil, nil, err
	} else if !configExists {
		return nil, nil, errors.New(message.ConfigNotFound)
	}

	// load config
	loadedConfig, err := configLoader.Load(configOptions, log)
	if err != nil {
		return nil, nil, err
	}

	// resolve dependencies
	dependencies, err := dependency.NewManager(loadedConfig, nil, configOptions, log).ResolveAll(dependency.ResolveOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("error resolving dependencies: %v", err)
	}

	return loadedConfig, dependencies, nil
}

func InterruptChan() (chan error, func()) {
	once := sync.Once{}
	c := make(chan error)
	return c, func() {
		once.Do(func() {
			close(c)
		})
	}
}

func SwitchDir(dir string) {
	err := os.Chdir(dir)
	ExpectNoError(err)
}

func CleanupTempDir(initialDir, tempDir string) {
	err := os.RemoveAll(tempDir)
	ExpectNoError(err)

	err = os.Chdir(initialDir)
	ExpectNoError(err)
}

func CopyToTempDir(relativePath string) (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	err = copy.Copy(relativePath, dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	err = os.Chdir(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}

func ChangeToTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	err = os.Chdir(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	return dir, nil
}
