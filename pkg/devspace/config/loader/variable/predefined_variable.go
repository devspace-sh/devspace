package variable

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/mitchellh/go-homedir"
)

// PredefinedVariableOptions holds the options for a predefined variable to load
type PredefinedVariableOptions struct {
	ConfigPath string
	KubeClient kubectl.Client
	Profile    string
}

// PredefinedVariableFunction is the definition of a predefined variable
type PredefinedVariableFunction func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error)

// predefinedVars holds all predefined variables that can be used in the config
var predefinedVars = map[string]PredefinedVariableFunction{
	"devspace.version": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		return upgrade.GetVersion(), nil
	},
	"devspace.random": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		return randutil.GenerateRandomString(6), nil
	},
	"devspace.profile": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		return options.Profile, nil
	},
	"devspace.userHome": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		homeDir, err := homedir.Dir()
		if err != nil {
			return nil, err
		}
		return homeDir, nil
	},
	"devspace.timestamp": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		return strconv.FormatInt(time.Now().Unix(), 10), nil
	},
	"devspace.git.branch": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		configPath := options.ConfigPath
		branch, err := git.GetBranch(filepath.Dir(configPath))
		if err != nil {
			return "", fmt.Errorf("error retrieving git branch: %v, but predefined var devspace.git.branch is used", err)
		}

		return branch, nil
	},
	"devspace.git.commit": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		configPath := options.ConfigPath
		hash, err := git.GetHash(ctx, filepath.Dir(configPath))
		if err != nil {
			return "", fmt.Errorf("no git repository found (%v), but predefined var devspace.git.commit is used", err)
		}

		return hash[:8], nil
	},
	"devspace.context": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		if options.KubeClient == nil {
			return "", nil
		}

		return options.KubeClient.CurrentContext(), nil
	},
	"devspace.namespace": func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
		if options.KubeClient == nil {
			return "", nil
		}

		return options.KubeClient.Namespace(), nil
	},
}

func init() {
	// migrate old names
	predefinedVars["DEVSPACE_VERSION"] = predefinedVars["devspace.version"]
	predefinedVars["DEVSPACE_RANDOM"] = predefinedVars["devspace.random"]
	predefinedVars["DEVSPACE_PROFILE"] = predefinedVars["devspace.profile"]
	predefinedVars["DEVSPACE_USER_HOME"] = predefinedVars["devspace.userHome"]
	predefinedVars["DEVSPACE_TIMESTAMP"] = predefinedVars["devspace.timestamp"]
	predefinedVars["DEVSPACE_GIT_BRANCH"] = predefinedVars["devspace.git.branch"]
	predefinedVars["DEVSPACE_GIT_COMMIT"] = predefinedVars["devspace.git.commit"]
	predefinedVars["DEVSPACE_CONTEXT"] = predefinedVars["devspace.context"]
	predefinedVars["DEVSPACE_NAMESPACE"] = predefinedVars["devspace.namespace"]
}

func IsPredefinedVariable(name string) bool {
	_, ok := predefinedVars[name]
	return ok
}

func AddPredefinedVars(plugins []plugin.Metadata) {
	for _, p := range plugins {
		pluginName := p.Name
		pluginFolder := p.PluginFolder
		for _, variable := range p.Vars {
			v := variable
			predefinedVars[variable.Name] = func(ctx context.Context, options *PredefinedVariableOptions) (interface{}, error) {
				args, err := json.Marshal(os.Args)
				if err != nil {
					return "", err
				}

				buffer := &bytes.Buffer{}
				err = plugin.CallPluginExecutable(filepath.Join(pluginFolder, plugin.PluginBinary), v.BaseArgs, map[string]string{
					plugin.OsArgsEnv: string(args),
				}, buffer)
				if err != nil {
					return "", fmt.Errorf("executing plugin %s: %s - %v", pluginName, buffer.String(), err)
				}

				return strings.TrimSpace(buffer.String()), nil
			}
		}
	}
}

// NewPredefinedVariable creates a new predefined variable for the given name or fails if there
// is none with the given name
func NewPredefinedVariable(name string, options *PredefinedVariableOptions) (Variable, error) {
	if _, ok := predefinedVars[name]; !ok {
		return nil, errors.New("predefined variable " + name + " not found")
	}

	return &predefinedVariable{
		name:    name,
		options: options,
	}, nil
}

type predefinedVariable struct {
	name    string
	options *PredefinedVariableOptions
}

func (p *predefinedVariable) Load(ctx context.Context, definition *latest.Variable) (interface{}, error) {
	getVar, ok := predefinedVars[p.name]
	if !ok {
		return nil, errors.New("predefined variable " + p.name + " not found")
	}

	return getVar(ctx, p.options)
}
