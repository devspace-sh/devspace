package loader

import (
	"bytes"
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/util"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/plugin"
	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/devspace-cloud/devspace/pkg/util/randutil"
)

// predefinedVars holds all predefined variables that can be used in the config
var predefinedVars = map[string]func(loader *configLoader) (string, error){
	"DEVSPACE_RANDOM": func(loader *configLoader) (string, error) {
		ret, err := randutil.GenerateRandomString(6)
		if err != nil {
			return "", err
		}

		return ret, nil
	},
	"DEVSPACE_TIMESTAMP": func(loader *configLoader) (string, error) {
		return strconv.FormatInt(time.Now().Unix(), 10), nil
	},
	"DEVSPACE_GIT_COMMIT": func(loader *configLoader) (string, error) {
		hash, err := git.GetHash(filepath.Dir(loader.ConfigPath()))
		if err != nil {
			return "", fmt.Errorf("No git repository found (%v), but predefined var DEVSPACE_GIT_COMMIT is used", err)
		}

		return hash[:8], nil
	},
	"DEVSPACE_CONTEXT": func(loader *configLoader) (string, error) {
		_, activeContext, _, err := util.NewClientByContext(loader.options.KubeContext, loader.options.Namespace, false, loader.kubeConfigLoader)
		if err != nil {
			return "", err
		}

		return activeContext, nil
	},
	"DEVSPACE_NAMESPACE": func(loader *configLoader) (string, error) {
		_, _, activeNamespace, err := util.NewClientByContext(loader.options.KubeContext, loader.options.Namespace, false, loader.kubeConfigLoader)
		if err != nil {
			return "", err
		}

		return activeNamespace, nil
	},
}

func AddPredefinedVars(plugins []plugin.Metadata) {
	for _, p := range plugins {
		pluginName := p.Name
		pluginFolder := p.PluginFolder
		for _, variable := range p.Vars {
			v := variable
			predefinedVars[variable.Name] = func(configLoader *configLoader) (string, error) {
				buffer := &bytes.Buffer{}
				err := plugin.CallPluginExecutable(filepath.Join(pluginFolder, plugin.PluginBinary), v.BaseArgs, map[string]string{
					"DEVSPACE_PLUGIN_KUBE_CONTEXT_FLAG":   configLoader.options.KubeContext,
					"DEVSPACE_PLUGIN_KUBE_NAMESPACE_FLAG": configLoader.options.Namespace,
				}, buffer)
				if err != nil {
					return "", fmt.Errorf("executing plugin %s: %s - %v", pluginName, buffer.String(), err)
				}

				return strings.TrimSpace(buffer.String()), nil
			}
		}
	}
}

func (l *configLoader) resolvePredefinedVar(name string) (bool, string, error) {
	name = strings.ToUpper(name)
	if getVar, ok := predefinedVars[name]; ok {
		if l.resolvedVars == nil {
			l.resolvedVars = map[string]string{}
		}

		val, ok := l.resolvedVars[name]
		if !ok {
			val, err := getVar(l)
			if err != nil {
				return false, "", err
			}

			l.resolvedVars[name] = val
			return true, val, nil
		}

		return true, val, nil
	}

	generatedConfig, err := l.Generated()
	if err != nil {
		return false, "", nil
	}
	// Load space domain environment variable
	if strings.HasPrefix(name, "DEVSPACE_SPACE_DOMAIN") {
		// Check if its in generated config
		if val, ok := generatedConfig.Vars[name]; ok {
			return true, val, nil
		}

		return true, name, nil
	}

	return false, "", nil
}
